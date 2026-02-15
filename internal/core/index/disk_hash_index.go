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
	hashMagic   = 0x48494458
	hashVersion = 1

	heapHeaderSize      = 10
	dirPages            = 64
	dirEntryBytes       = 4
	dirEntriesPerPage   = (page.PageSize - heapHeaderSize) / dirEntryBytes
	dataStartPage       = 1 + dirPages
	maxLoadFactor       = 0.75
	targetBucketEntries = 64

	bucketHdrNextOverflowOff = heapHeaderSize
	bucketHdrEntryCountOff   = heapHeaderSize + 4
	bucketHdrFreeOff         = heapHeaderSize + 8
	bucketHdrSize            = 12
	bucketDataStartOff       = heapHeaderSize + bucketHdrSize

	metaMagicOff       = heapHeaderSize
	metaVersionOff     = heapHeaderSize + 4
	metaBucketCountOff = heapHeaderSize + 8
	metaLowmaskOff     = heapHeaderSize + 12
	metaHighmaskOff    = heapHeaderSize + 16
	metaSplitPtrOff    = heapHeaderSize + 20
	metaMaxBucketOff   = heapHeaderSize + 24
	metaRecordCountOff = heapHeaderSize + 28
	metaNextPageIDOff  = heapHeaderSize + 36
)

var (
	ErrNotHashIndex       = errors.New("not a hash index")
	ErrBadMagic           = errors.New("bad magic number")
	ErrUnsupportedVersion = errors.New("unsupported version")
	ErrRangeNotSupported  = errors.New("hash index does not support range search")
)

type hashMeta struct {
	bucketCount  int32
	lowmask      int32
	highmask     int32
	splitPointer int32
	maxBucket    int32
	recordCount  int64
	nextPageID   int32
}

type hashEntry struct {
	hash int32
	key  any
	tid  TID
}

type DiskHashIndex struct {
	mu         sync.Mutex
	root       string
	bufferPool buffer.BufferPoolManager
	catalog    manager.CatalogManager
	def        *model.IndexDefinition
	fileID     string
	keyType    *model.TypeDefinition
	meta       hashMeta
}

func NewDiskHashIndex(root string, bufferPool buffer.BufferPoolManager, catalog manager.CatalogManager, def *model.IndexDefinition) (*DiskHashIndex, error) {
	if def.IndexType() != types.IndexTypeHash {
		return nil, ErrNotHashIndex
	}

	keyType, err := catalog.GetTypeByOid(def.KeyTypeOid())
	if err != nil || keyType == nil {
		return nil, errors.New("unknown key type")
	}

	idx := &DiskHashIndex{
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

func (idx *DiskHashIndex) Definition() *model.IndexDefinition {
	return idx.def
}

func (idx *DiskHashIndex) Insert(key any, tid TID) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	hash := idx.hashFunction(key)
	bucketID := idx.computeBucket(hash)
	entry := hashEntry{hash: hash, key: key, tid: tid}

	if err := idx.insertIntoBucketChain(bucketID, entry); err != nil {
		return err
	}

	idx.meta.recordCount++
	if err := idx.writeMeta(); err != nil {
		return err
	}

	for idx.loadFactor() > maxLoadFactor {
		if err := idx.performSplit(); err != nil {
			return err
		}
	}

	return nil
}

func (idx *DiskHashIndex) Search(key any) ([]TID, error) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	hash := idx.hashFunction(key)
	bucketID := idx.computeBucket(hash)

	var result []TID
	err := idx.forEachEntry(bucketID, func(e hashEntry) {
		if e.hash == hash && idx.compareKeys(key, e.key) == 0 {
			result = append(result, e.tid)
		}
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (idx *DiskHashIndex) RangeSearch(from any, fromInclusive bool, to any, toInclusive bool) ([]TID, error) {
	return nil, ErrRangeNotSupported
}

func (idx *DiskHashIndex) initOrLoad() error {
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

func (idx *DiskHashIndex) initializeNew() error {
	metaPage := page.NewHeapPage(0)
	if _, err := idx.bufferPool.NewPage(idx.key(0), metaPage); err != nil {
		return err
	}

	for i := 1; i <= dirPages; i++ {
		dirPage := page.NewHeapPage(i)
		if _, err := idx.bufferPool.NewPage(idx.key(i), dirPage); err != nil {
			return err
		}
	}

	initialBuckets := int32(16)
	idx.meta = hashMeta{
		bucketCount:  initialBuckets,
		lowmask:      initialBuckets - 1,
		highmask:     initialBuckets - 1,
		splitPointer: 0,
		maxBucket:    initialBuckets - 1,
		recordCount:  0,
		nextPageID:   dataStartPage,
	}

	for bucketID := int32(0); bucketID < initialBuckets; bucketID++ {
		headPageID, err := idx.allocateDataPage()
		if err != nil {
			return err
		}
		if err := idx.initBucketPage(headPageID); err != nil {
			return err
		}
		if err := idx.setBucketHeadPageID(bucketID, headPageID); err != nil {
			return err
		}
	}

	return idx.writeMeta()
}

func (idx *DiskHashIndex) loadMeta() error {
	slot, err := idx.bufferPool.GetPage(idx.key(0))
	if err != nil {
		return err
	}
	buf := slot.Page().Bytes()

	magic := int32(binary.BigEndian.Uint32(buf[metaMagicOff:]))
	if magic != hashMagic {
		return ErrBadMagic
	}

	version := int32(binary.BigEndian.Uint32(buf[metaVersionOff:]))
	if version != hashVersion {
		return ErrUnsupportedVersion
	}

	idx.meta = hashMeta{
		bucketCount:  int32(binary.BigEndian.Uint32(buf[metaBucketCountOff:])),
		lowmask:      int32(binary.BigEndian.Uint32(buf[metaLowmaskOff:])),
		highmask:     int32(binary.BigEndian.Uint32(buf[metaHighmaskOff:])),
		splitPointer: int32(binary.BigEndian.Uint32(buf[metaSplitPtrOff:])),
		maxBucket:    int32(binary.BigEndian.Uint32(buf[metaMaxBucketOff:])),
		recordCount:  int64(binary.BigEndian.Uint64(buf[metaRecordCountOff:])),
		nextPageID:   int32(binary.BigEndian.Uint32(buf[metaNextPageIDOff:])),
	}

	return nil
}

func (idx *DiskHashIndex) writeMeta() error {
	slot, err := idx.bufferPool.GetPage(idx.key(0))
	if err != nil {
		return err
	}
	buf := slot.Page().Bytes()

	binary.BigEndian.PutUint32(buf[metaMagicOff:], uint32(hashMagic))
	binary.BigEndian.PutUint32(buf[metaVersionOff:], uint32(hashVersion))
	binary.BigEndian.PutUint32(buf[metaBucketCountOff:], uint32(idx.meta.bucketCount))
	binary.BigEndian.PutUint32(buf[metaLowmaskOff:], uint32(idx.meta.lowmask))
	binary.BigEndian.PutUint32(buf[metaHighmaskOff:], uint32(idx.meta.highmask))
	binary.BigEndian.PutUint32(buf[metaSplitPtrOff:], uint32(idx.meta.splitPointer))
	binary.BigEndian.PutUint32(buf[metaMaxBucketOff:], uint32(idx.meta.maxBucket))
	binary.BigEndian.PutUint64(buf[metaRecordCountOff:], uint64(idx.meta.recordCount))
	binary.BigEndian.PutUint32(buf[metaNextPageIDOff:], uint32(idx.meta.nextPageID))

	return idx.bufferPool.UpdatePage(idx.key(0), slot.Page())
}

func (idx *DiskHashIndex) hashFunction(key any) int32 {
	switch k := key.(type) {
	case int64:
		return int32(k^(k>>32)) & 0x7fffffff
	case string:
		h := int32(0)
		for _, c := range k {
			h = 31*h + int32(c)
		}
		return h & 0x7fffffff
	default:
		return 0
	}
}

func (idx *DiskHashIndex) computeBucket(hash int32) int32 {
	bucket := hash & idx.meta.highmask
	if bucket > idx.meta.maxBucket {
		bucket = hash & idx.meta.lowmask
	}
	return bucket
}

func (idx *DiskHashIndex) loadFactor() float64 {
	buckets := idx.meta.bucketCount
	if buckets == 0 {
		buckets = 1
	}
	return float64(idx.meta.recordCount) / float64(buckets*targetBucketEntries)
}

func (idx *DiskHashIndex) performSplit() error {
	splitBucketID := idx.meta.splitPointer
	newBucketID := idx.meta.maxBucket + 1

	newHeadPageID, err := idx.allocateDataPage()
	if err != nil {
		return err
	}
	if err := idx.initBucketPage(newHeadPageID); err != nil {
		return err
	}
	if err := idx.setBucketHeadPageID(newBucketID, newHeadPageID); err != nil {
		return err
	}

	newMaxBucket := idx.meta.maxBucket + 1
	newBucketCount := newMaxBucket + 1
	newHighmask := idx.meta.highmask
	if newMaxBucket > newHighmask {
		newHighmask = (newHighmask << 1) | 1
	}

	idx.meta.bucketCount = newBucketCount
	idx.meta.maxBucket = newMaxBucket
	idx.meta.highmask = newHighmask
	if err := idx.writeMeta(); err != nil {
		return err
	}

	entries, err := idx.drainBucketChain(splitBucketID)
	if err != nil {
		return err
	}

	for _, e := range entries {
		target := idx.computeBucket(e.hash)
		if err := idx.insertIntoBucketChain(target, e); err != nil {
			return err
		}
	}

	nextSplitPointer := idx.meta.splitPointer + 1
	newLowmask := idx.meta.lowmask
	if nextSplitPointer == idx.meta.lowmask+1 {
		newLowmask = idx.meta.highmask
		nextSplitPointer = 0
	}

	idx.meta.splitPointer = nextSplitPointer
	idx.meta.lowmask = newLowmask
	return idx.writeMeta()
}

func (idx *DiskHashIndex) allocateDataPage() (int32, error) {
	pageID := idx.meta.nextPageID
	idx.meta.nextPageID++

	newPage := page.NewHeapPage(int(pageID))
	if _, err := idx.bufferPool.NewPage(idx.key(int(pageID)), newPage); err != nil {
		return 0, err
	}

	return pageID, nil
}

func (idx *DiskHashIndex) initBucketPage(pageID int32) error {
	slot, err := idx.bufferPool.GetPage(idx.key(int(pageID)))
	if err != nil {
		return err
	}
	buf := slot.Page().Bytes()

	binary.BigEndian.PutUint32(buf[bucketHdrNextOverflowOff:], 0xFFFFFFFF)
	binary.BigEndian.PutUint32(buf[bucketHdrEntryCountOff:], 0)
	binary.BigEndian.PutUint32(buf[bucketHdrFreeOff:], uint32(bucketDataStartOff))

	return idx.bufferPool.UpdatePage(idx.key(int(pageID)), slot.Page())
}

func (idx *DiskHashIndex) getBucketHeadPageID(bucketID int32) (int32, error) {
	dirPageIdx := int(bucketID / dirEntriesPerPage)
	entryIdx := int(bucketID % dirEntriesPerPage)
	dirPageID := 1 + dirPageIdx

	slot, err := idx.bufferPool.GetPage(idx.key(dirPageID))
	if err != nil {
		return 0, err
	}
	buf := slot.Page().Bytes()

	off := heapHeaderSize + entryIdx*dirEntryBytes
	return int32(binary.BigEndian.Uint32(buf[off:])), nil
}

func (idx *DiskHashIndex) setBucketHeadPageID(bucketID int32, headPageID int32) error {
	dirPageIdx := int(bucketID / dirEntriesPerPage)
	entryIdx := int(bucketID % dirEntriesPerPage)
	dirPageID := 1 + dirPageIdx

	slot, err := idx.bufferPool.GetPage(idx.key(dirPageID))
	if err != nil {
		return err
	}
	buf := slot.Page().Bytes()

	off := heapHeaderSize + entryIdx*dirEntryBytes
	binary.BigEndian.PutUint32(buf[off:], uint32(headPageID))

	return idx.bufferPool.UpdatePage(idx.key(dirPageID), slot.Page())
}

func (idx *DiskHashIndex) insertIntoBucketChain(bucketID int32, entry hashEntry) error {
	current, err := idx.getBucketHeadPageID(bucketID)
	if err != nil {
		return err
	}

	entryBytes := idx.encodeEntry(entry)

	for {
		slot, err := idx.bufferPool.GetPage(idx.key(int(current)))
		if err != nil {
			return err
		}
		buf := slot.Page().Bytes()

		free := int(binary.BigEndian.Uint32(buf[bucketHdrFreeOff:]))
		if free+len(entryBytes) <= page.PageSize {
			copy(buf[free:], entryBytes)
			binary.BigEndian.PutUint32(buf[bucketHdrFreeOff:], uint32(free+len(entryBytes)))
			cnt := binary.BigEndian.Uint32(buf[bucketHdrEntryCountOff:])
			binary.BigEndian.PutUint32(buf[bucketHdrEntryCountOff:], cnt+1)
			return idx.bufferPool.UpdatePage(idx.key(int(current)), slot.Page())
		}

		next := int32(binary.BigEndian.Uint32(buf[bucketHdrNextOverflowOff:]))
		if next == -1 {
			overflowPageID, err := idx.allocateDataPage()
			if err != nil {
				return err
			}
			if err := idx.initBucketPage(overflowPageID); err != nil {
				return err
			}
			binary.BigEndian.PutUint32(buf[bucketHdrNextOverflowOff:], uint32(overflowPageID))
			if err := idx.bufferPool.UpdatePage(idx.key(int(current)), slot.Page()); err != nil {
				return err
			}
			current = overflowPageID
		} else {
			current = next
		}
	}
}

func (idx *DiskHashIndex) drainBucketChain(bucketID int32) ([]hashEntry, error) {
	head, err := idx.getBucketHeadPageID(bucketID)
	if err != nil {
		return nil, err
	}

	var entries []hashEntry
	current := head

	for current != -1 && current != 0 {
		slot, err := idx.bufferPool.GetPage(idx.key(int(current)))
		if err != nil {
			return nil, err
		}
		buf := slot.Page().Bytes()

		pageEntries, err := idx.readAllEntriesFromPage(buf)
		if err != nil {
			return nil, err
		}
		entries = append(entries, pageEntries...)

		next := int32(binary.BigEndian.Uint32(buf[bucketHdrNextOverflowOff:]))
		current = next
	}

	if err := idx.initBucketPage(head); err != nil {
		return nil, err
	}

	return entries, nil
}

func (idx *DiskHashIndex) forEachEntry(bucketID int32, fn func(hashEntry)) error {
	head, err := idx.getBucketHeadPageID(bucketID)
	if err != nil {
		return err
	}

	current := head
	for current != -1 && current != 0 {
		slot, err := idx.bufferPool.GetPage(idx.key(int(current)))
		if err != nil {
			return err
		}
		buf := slot.Page().Bytes()

		entries, err := idx.readAllEntriesFromPage(buf)
		if err != nil {
			return err
		}
		for _, e := range entries {
			fn(e)
		}

		next := int32(binary.BigEndian.Uint32(buf[bucketHdrNextOverflowOff:]))
		current = next
	}

	return nil
}

func (idx *DiskHashIndex) readAllEntriesFromPage(buf []byte) ([]hashEntry, error) {
	cnt := int(binary.BigEndian.Uint32(buf[bucketHdrEntryCountOff:]))
	var entries []hashEntry
	off := bucketDataStartOff

	for i := 0; i < cnt; i++ {
		entry, nextOff, err := idx.decodeEntry(buf, off)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
		off = nextOff
	}

	return entries, nil
}

func (idx *DiskHashIndex) encodeEntry(entry hashEntry) []byte {
	switch idx.keyType.Name() {
	case "INT64":
		buf := make([]byte, 4+8+TIDSize)
		binary.BigEndian.PutUint32(buf[0:], uint32(entry.hash))
		binary.BigEndian.PutUint64(buf[4:], uint64(entry.key.(int64)))
		binary.BigEndian.PutUint32(buf[12:], uint32(entry.tid.pageID))
		binary.BigEndian.PutUint16(buf[16:], uint16(entry.tid.slotID))
		return buf
	case "VARCHAR":
		s := entry.key.(string)
		buf := make([]byte, 4+2+len(s)+TIDSize)
		binary.BigEndian.PutUint32(buf[0:], uint32(entry.hash))
		binary.BigEndian.PutUint16(buf[4:], uint16(len(s)))
		copy(buf[6:], s)
		off := 6 + len(s)
		binary.BigEndian.PutUint32(buf[off:], uint32(entry.tid.pageID))
		binary.BigEndian.PutUint16(buf[off+4:], uint16(entry.tid.slotID))
		return buf
	}
	return nil
}

func (idx *DiskHashIndex) decodeEntry(buf []byte, off int) (hashEntry, int, error) {
	hash := int32(binary.BigEndian.Uint32(buf[off:]))
	off += 4

	var key any
	switch idx.keyType.Name() {
	case "INT64":
		key = int64(binary.BigEndian.Uint64(buf[off:]))
		off += 8
	case "VARCHAR":
		strLen := int(binary.BigEndian.Uint16(buf[off:]))
		off += 2
		key = string(buf[off : off+strLen])
		off += strLen
	}

	tid := TID{
		pageID: int32(binary.BigEndian.Uint32(buf[off:])),
		slotID: int16(binary.BigEndian.Uint16(buf[off+4:])),
	}
	off += TIDSize

	return hashEntry{hash: hash, key: key, tid: tid}, off, nil
}

func (idx *DiskHashIndex) compareKeys(a, b any) int {
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

func (idx *DiskHashIndex) key(pageID int) memmodel.PageKey {
	return memmodel.NewPageKey(idx.fileID, pageID)
}
