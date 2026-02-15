package index

import (
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"GopherDB/internal/core/catalog/manager"
	"GopherDB/internal/core/catalog/model"
	"GopherDB/internal/core/memory/buffer"
	memmodel "GopherDB/internal/core/memory/model"
	"GopherDB/internal/core/memory/page"
	"GopherDB/internal/core/types"
)

const (
	btreeMagic   = 0x42494458
	btreeVersion = 1
	nodeMagic    = 0x424E4F44

	btreeHeapHeaderSize = 10
	btreePageCapacity   = page.PageSize - btreeHeapHeaderSize

	btreeMetaMagicOff        = btreeHeapHeaderSize
	btreeMetaVersionOff      = btreeHeapHeaderSize + 4
	btreeMetaRootOff         = btreeHeapHeaderSize + 8
	btreeMetaHeightOff       = btreeHeapHeaderSize + 12
	btreeMetaLeftmostLeafOff = btreeHeapHeaderSize + 16
	btreeMetaNextPageIDOff   = btreeHeapHeaderSize + 20

	nodeIsLeafOff    = btreeHeapHeaderSize + 4
	nodeParentOff    = btreeHeapHeaderSize + 8
	nodeLeftSibOff   = btreeHeapHeaderSize + 12
	nodeRightSibOff  = btreeHeapHeaderSize + 16
	nodeKeyCountOff  = btreeHeapHeaderSize + 20
	nodeHdrSize      = 24
	nodeDataStartOff = btreeHeapHeaderSize + nodeHdrSize
)

var (
	ErrNotBTreeIndex = errors.New("not a btree index")
)

type btreeMeta struct {
	rootPageID     int32
	height         int32
	leftmostLeafID int32
	nextPageID     int32
}

type btreeNode struct {
	pageID         int32
	isLeaf         bool
	parentPageID   int32
	leftSibPageID  int32
	rightSibPageID int32
	keys           []any
	values         [][]TID
	children       []int32
}

type DiskBTreeIndex struct {
	mu         sync.Mutex
	root       string
	bufferPool buffer.BufferPoolManager
	catalog    manager.CatalogManager
	def        *model.IndexDefinition
	fileID     string
	keyType    *model.TypeDefinition
	meta       btreeMeta
}

func NewDiskBTreeIndex(root string, bufferPool buffer.BufferPoolManager, catalog manager.CatalogManager, def *model.IndexDefinition) (*DiskBTreeIndex, error) {
	if def.IndexType() != types.IndexTypeBTree {
		return nil, ErrNotBTreeIndex
	}

	keyType, err := catalog.GetTypeByOid(def.KeyTypeOid())
	if err != nil || keyType == nil {
		return nil, errors.New("unknown key type")
	}

	idx := &DiskBTreeIndex{
		root:       root,
		bufferPool: bufferPool,
		catalog:    catalog,
		def:        def,
		fileID:     def.FileNode(),
		keyType:    keyType,
	}

	if err := idx.initOrLoad(); err != nil {
		return nil, err
	}

	return idx, nil
}

func (idx *DiskBTreeIndex) Definition() *model.IndexDefinition {
	return idx.def
}

func (idx *DiskBTreeIndex) Insert(key any, tid TID) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	var path []int32
	leaf := idx.findLeaf(key, &path)

	idx.insertIntoLeaf(leaf, key, tid)

	if idx.estimateSize(leaf) <= btreePageCapacity {
		return idx.writeNode(leaf)
	}

	return idx.splitLeaf(leaf, path)
}

func (idx *DiskBTreeIndex) Search(key any) ([]TID, error) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	leaf := idx.findLeaf(key, nil)
	pos := idx.lowerBound(leaf.keys, key)

	if pos >= len(leaf.keys) || idx.compareKeys(leaf.keys[pos], key) != 0 {
		return nil, nil
	}

	return leaf.values[pos], nil
}

func (idx *DiskBTreeIndex) RangeSearch(from any, fromInclusive bool, to any, toInclusive bool) ([]TID, error) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if from != nil && to != nil && idx.compareKeys(from, to) > 0 {
		return nil, nil
	}

	var leafPageID int32
	var startPos int

	if from == nil {
		leafPageID = idx.meta.leftmostLeafID
		startPos = 0
	} else {
		leaf := idx.findLeaf(from, nil)
		leafPageID = leaf.pageID
		startPos = idx.lowerBound(leaf.keys, from)
	}

	var result []TID
	currentLeafID := leafPageID
	pos := startPos

	for currentLeafID != -1 {
		leaf, err := idx.readNode(currentLeafID)
		if err != nil {
			return nil, err
		}

		for pos < len(leaf.keys) {
			k := leaf.keys[pos]

			if from != nil {
				cmp := idx.compareKeys(k, from)
				if cmp < 0 || (!fromInclusive && cmp == 0) {
					pos++
					continue
				}
			}

			if to != nil {
				cmp := idx.compareKeys(k, to)
				if cmp > 0 || (!toInclusive && cmp == 0) {
					return result, nil
				}
			}

			result = append(result, leaf.values[pos]...)
			pos++
		}

		currentLeafID = leaf.rightSibPageID
		pos = 0
	}

	return result, nil
}

func (idx *DiskBTreeIndex) initOrLoad() error {
	if err := os.MkdirAll(idx.root, 0755); err != nil {
		return err
	}

	filePath := filepath.Join(idx.root, idx.fileID)
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) || (err == nil && info.Size() == 0) {
		return idx.initializeNew()
	}

	return idx.loadMeta()
}

func (idx *DiskBTreeIndex) initializeNew() error {
	metaPage := page.NewHeapPage(0)
	if _, err := idx.bufferPool.NewPage(idx.key(0), metaPage); err != nil {
		return err
	}

	rootLeaf := page.NewHeapPage(1)
	if _, err := idx.bufferPool.NewPage(idx.key(1), rootLeaf); err != nil {
		return err
	}

	idx.meta = btreeMeta{
		rootPageID:     1,
		height:         1,
		leftmostLeafID: 1,
		nextPageID:     2,
	}

	leaf := &btreeNode{
		pageID:         1,
		isLeaf:         true,
		parentPageID:   -1,
		leftSibPageID:  -1,
		rightSibPageID: -1,
	}

	if err := idx.writeNode(leaf); err != nil {
		return err
	}

	return idx.writeMeta()
}

func (idx *DiskBTreeIndex) loadMeta() error {
	slot, err := idx.bufferPool.GetPage(idx.key(0))
	if err != nil {
		return err
	}
	buf := slot.Page().Bytes()

	magic := int32(binary.BigEndian.Uint32(buf[btreeMetaMagicOff:]))
	if magic != btreeMagic {
		return ErrBadMagic
	}

	version := int32(binary.BigEndian.Uint32(buf[btreeMetaVersionOff:]))
	if version != btreeVersion {
		return ErrUnsupportedVersion
	}

	idx.meta = btreeMeta{
		rootPageID:     int32(binary.BigEndian.Uint32(buf[btreeMetaRootOff:])),
		height:         int32(binary.BigEndian.Uint32(buf[btreeMetaHeightOff:])),
		leftmostLeafID: int32(binary.BigEndian.Uint32(buf[btreeMetaLeftmostLeafOff:])),
		nextPageID:     int32(binary.BigEndian.Uint32(buf[btreeMetaNextPageIDOff:])),
	}

	return nil
}

func (idx *DiskBTreeIndex) writeMeta() error {
	slot, err := idx.bufferPool.GetPage(idx.key(0))
	if err != nil {
		return err
	}
	buf := slot.Page().Bytes()

	binary.BigEndian.PutUint32(buf[btreeMetaMagicOff:], uint32(btreeMagic))
	binary.BigEndian.PutUint32(buf[btreeMetaVersionOff:], uint32(btreeVersion))
	binary.BigEndian.PutUint32(buf[btreeMetaRootOff:], uint32(idx.meta.rootPageID))
	binary.BigEndian.PutUint32(buf[btreeMetaHeightOff:], uint32(idx.meta.height))
	binary.BigEndian.PutUint32(buf[btreeMetaLeftmostLeafOff:], uint32(idx.meta.leftmostLeafID))
	binary.BigEndian.PutUint32(buf[btreeMetaNextPageIDOff:], uint32(idx.meta.nextPageID))

	return idx.bufferPool.UpdatePage(idx.key(0), slot.Page())
}

func (idx *DiskBTreeIndex) findLeaf(key any, path *[]int32) *btreeNode {
	current := idx.meta.rootPageID

	for {
		node, _ := idx.readNode(current)
		if path != nil {
			*path = append(*path, current)
		}

		if node.isLeaf {
			return node
		}

		pos := idx.upperBound(node.keys, key)
		current = node.children[pos]
	}
}

func (idx *DiskBTreeIndex) readNode(pageID int32) (*btreeNode, error) {
	slot, err := idx.bufferPool.GetPage(idx.key(int(pageID)))
	if err != nil {
		return nil, err
	}
	buf := slot.Page().Bytes()

	node := &btreeNode{
		pageID:         pageID,
		isLeaf:         buf[nodeIsLeafOff] == 1,
		parentPageID:   int32(binary.BigEndian.Uint32(buf[nodeParentOff:])),
		leftSibPageID:  int32(binary.BigEndian.Uint32(buf[nodeLeftSibOff:])),
		rightSibPageID: int32(binary.BigEndian.Uint32(buf[nodeRightSibOff:])),
	}

	keyCount := int(binary.BigEndian.Uint32(buf[nodeKeyCountOff:]))
	off := nodeDataStartOff

	for i := 0; i < keyCount; i++ {
		key, newOff := idx.decodeKey(buf, off)
		node.keys = append(node.keys, key)
		off = newOff
	}

	if node.isLeaf {
		for i := 0; i < keyCount; i++ {
			tidCount := int(binary.BigEndian.Uint32(buf[off:]))
			off += 4
			var tids []TID
			for j := 0; j < tidCount; j++ {
				tid := TID{
					pageID: int32(binary.BigEndian.Uint32(buf[off:])),
					slotID: int16(binary.BigEndian.Uint16(buf[off+4:])),
				}
				tids = append(tids, tid)
				off += TIDSize
			}
			node.values = append(node.values, tids)
		}
	} else {
		childCount := keyCount + 1
		for i := 0; i < childCount; i++ {
			child := int32(binary.BigEndian.Uint32(buf[off:]))
			node.children = append(node.children, child)
			off += 4
		}
	}

	return node, nil
}

func (idx *DiskBTreeIndex) writeNode(node *btreeNode) error {
	slot, err := idx.bufferPool.GetPage(idx.key(int(node.pageID)))
	if err != nil {
		return err
	}
	buf := slot.Page().Bytes()

	binary.BigEndian.PutUint32(buf[btreeHeapHeaderSize:], uint32(nodeMagic))
	if node.isLeaf {
		buf[nodeIsLeafOff] = 1
	} else {
		buf[nodeIsLeafOff] = 0
	}
	binary.BigEndian.PutUint32(buf[nodeParentOff:], uint32(node.parentPageID))
	binary.BigEndian.PutUint32(buf[nodeLeftSibOff:], uint32(node.leftSibPageID))
	binary.BigEndian.PutUint32(buf[nodeRightSibOff:], uint32(node.rightSibPageID))
	binary.BigEndian.PutUint32(buf[nodeKeyCountOff:], uint32(len(node.keys)))

	off := nodeDataStartOff

	for _, key := range node.keys {
		off = idx.encodeKey(buf, off, key)
	}

	if node.isLeaf {
		for _, tids := range node.values {
			binary.BigEndian.PutUint32(buf[off:], uint32(len(tids)))
			off += 4
			for _, tid := range tids {
				binary.BigEndian.PutUint32(buf[off:], uint32(tid.pageID))
				binary.BigEndian.PutUint16(buf[off+4:], uint16(tid.slotID))
				off += TIDSize
			}
		}
	} else {
		for _, child := range node.children {
			binary.BigEndian.PutUint32(buf[off:], uint32(child))
			off += 4
		}
	}

	return idx.bufferPool.UpdatePage(idx.key(int(node.pageID)), slot.Page())
}

func (idx *DiskBTreeIndex) insertIntoLeaf(leaf *btreeNode, key any, tid TID) {
	pos := idx.lowerBound(leaf.keys, key)

	if pos < len(leaf.keys) && idx.compareKeys(leaf.keys[pos], key) == 0 {
		leaf.values[pos] = append(leaf.values[pos], tid)
		return
	}

	leaf.keys = append(leaf.keys, nil)
	copy(leaf.keys[pos+1:], leaf.keys[pos:])
	leaf.keys[pos] = key

	leaf.values = append(leaf.values, nil)
	copy(leaf.values[pos+1:], leaf.values[pos:])
	leaf.values[pos] = []TID{tid}
}

func (idx *DiskBTreeIndex) splitLeaf(leaf *btreeNode, path []int32) error {
	mid := len(leaf.keys) / 2

	newPageID := idx.meta.nextPageID
	idx.meta.nextPageID++

	newPage := page.NewHeapPage(int(newPageID))
	if _, err := idx.bufferPool.NewPage(idx.key(int(newPageID)), newPage); err != nil {
		return err
	}

	newLeaf := &btreeNode{
		pageID:         newPageID,
		isLeaf:         true,
		parentPageID:   leaf.parentPageID,
		leftSibPageID:  leaf.pageID,
		rightSibPageID: leaf.rightSibPageID,
		keys:           append([]any{}, leaf.keys[mid:]...),
		values:         append([][]TID{}, leaf.values[mid:]...),
	}

	leaf.keys = leaf.keys[:mid]
	leaf.values = leaf.values[:mid]
	leaf.rightSibPageID = newPageID

	if err := idx.writeNode(leaf); err != nil {
		return err
	}
	if err := idx.writeNode(newLeaf); err != nil {
		return err
	}

	if newLeaf.rightSibPageID != -1 {
		rightSib, err := idx.readNode(newLeaf.rightSibPageID)
		if err != nil {
			return err
		}
		rightSib.leftSibPageID = newPageID
		if err := idx.writeNode(rightSib); err != nil {
			return err
		}
	}

	splitKey := newLeaf.keys[0]
	return idx.insertIntoParent(leaf, splitKey, newLeaf, path)
}

func (idx *DiskBTreeIndex) insertIntoParent(left *btreeNode, key any, right *btreeNode, path []int32) error {
	if left.parentPageID == -1 {
		return idx.createNewRoot(left, key, right)
	}

	parent, err := idx.readNode(left.parentPageID)
	if err != nil {
		return err
	}

	pos := idx.upperBound(parent.keys, key)

	parent.keys = append(parent.keys, nil)
	copy(parent.keys[pos+1:], parent.keys[pos:])
	parent.keys[pos] = key

	parent.children = append(parent.children, 0)
	copy(parent.children[pos+2:], parent.children[pos+1:])
	parent.children[pos+1] = right.pageID

	right.parentPageID = parent.pageID
	if err := idx.writeNode(right); err != nil {
		return err
	}

	if idx.estimateSize(parent) <= btreePageCapacity {
		return idx.writeNode(parent)
	}

	return idx.splitInternal(parent, path[:len(path)-1])
}

func (idx *DiskBTreeIndex) createNewRoot(left *btreeNode, key any, right *btreeNode) error {
	newRootID := idx.meta.nextPageID
	idx.meta.nextPageID++

	newPage := page.NewHeapPage(int(newRootID))
	if _, err := idx.bufferPool.NewPage(idx.key(int(newRootID)), newPage); err != nil {
		return err
	}

	newRoot := &btreeNode{
		pageID:         newRootID,
		isLeaf:         false,
		parentPageID:   -1,
		leftSibPageID:  -1,
		rightSibPageID: -1,
		keys:           []any{key},
		children:       []int32{left.pageID, right.pageID},
	}

	left.parentPageID = newRootID
	right.parentPageID = newRootID

	if err := idx.writeNode(newRoot); err != nil {
		return err
	}
	if err := idx.writeNode(left); err != nil {
		return err
	}
	if err := idx.writeNode(right); err != nil {
		return err
	}

	idx.meta.rootPageID = newRootID
	idx.meta.height++
	return idx.writeMeta()
}

func (idx *DiskBTreeIndex) splitInternal(node *btreeNode, path []int32) error {
	mid := len(node.keys) / 2
	splitKey := node.keys[mid]

	newPageID := idx.meta.nextPageID
	idx.meta.nextPageID++

	newPage := page.NewHeapPage(int(newPageID))
	if _, err := idx.bufferPool.NewPage(idx.key(int(newPageID)), newPage); err != nil {
		return err
	}

	newNode := &btreeNode{
		pageID:         newPageID,
		isLeaf:         false,
		parentPageID:   node.parentPageID,
		leftSibPageID:  node.pageID,
		rightSibPageID: node.rightSibPageID,
		keys:           append([]any{}, node.keys[mid+1:]...),
		children:       append([]int32{}, node.children[mid+1:]...),
	}

	node.keys = node.keys[:mid]
	node.children = node.children[:mid+1]
	node.rightSibPageID = newPageID

	for _, childID := range newNode.children {
		child, err := idx.readNode(childID)
		if err != nil {
			return err
		}
		child.parentPageID = newPageID
		if err := idx.writeNode(child); err != nil {
			return err
		}
	}

	if err := idx.writeNode(node); err != nil {
		return err
	}
	if err := idx.writeNode(newNode); err != nil {
		return err
	}

	return idx.insertIntoParent(node, splitKey, newNode, path)
}

func (idx *DiskBTreeIndex) estimateSize(node *btreeNode) int {
	size := nodeDataStartOff

	for _, key := range node.keys {
		size += idx.keySize(key)
	}

	if node.isLeaf {
		for _, tids := range node.values {
			size += 4 + len(tids)*TIDSize
		}
	} else {
		size += len(node.children) * 4
	}

	return size
}

func (idx *DiskBTreeIndex) keySize(key any) int {
	switch idx.keyType.Name() {
	case "INT64":
		return 8
	case "VARCHAR":
		return 2 + len(key.(string))
	}
	return 0
}

func (idx *DiskBTreeIndex) encodeKey(buf []byte, off int, key any) int {
	switch idx.keyType.Name() {
	case "INT64":
		binary.BigEndian.PutUint64(buf[off:], uint64(key.(int64)))
		return off + 8
	case "VARCHAR":
		s := key.(string)
		binary.BigEndian.PutUint16(buf[off:], uint16(len(s)))
		copy(buf[off+2:], s)
		return off + 2 + len(s)
	}
	return off
}

func (idx *DiskBTreeIndex) decodeKey(buf []byte, off int) (any, int) {
	switch idx.keyType.Name() {
	case "INT64":
		return int64(binary.BigEndian.Uint64(buf[off:])), off + 8
	case "VARCHAR":
		strLen := int(binary.BigEndian.Uint16(buf[off:]))
		return string(buf[off+2 : off+2+strLen]), off + 2 + strLen
	}
	return nil, off
}

func (idx *DiskBTreeIndex) lowerBound(keys []any, key any) int {
	lo, hi := 0, len(keys)
	for lo < hi {
		mid := (lo + hi) / 2
		if idx.compareKeys(keys[mid], key) < 0 {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}

func (idx *DiskBTreeIndex) upperBound(keys []any, key any) int {
	lo, hi := 0, len(keys)
	for lo < hi {
		mid := (lo + hi) / 2
		if idx.compareKeys(keys[mid], key) <= 0 {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}

func (idx *DiskBTreeIndex) compareKeys(a, b any) int {
	switch idx.keyType.Name() {
	case "INT64":
		av, bv := a.(int64), b.(int64)
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
		return 0
	case "VARCHAR":
		av, bv := a.(string), b.(string)
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
		return 0
	}
	return 0
}

func (idx *DiskBTreeIndex) key(pageID int) memmodel.PageKey {
	return memmodel.NewPageKey(idx.fileID, pageID)
}

