package server

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"GopherDB/internal/core/catalog/manager"
	"GopherDB/internal/core/engine"
	"GopherDB/internal/core/index"
	"GopherDB/internal/core/memory/buffer"
	"GopherDB/internal/core/sql/lexer"
	"GopherDB/internal/core/sql/semantic"
	"GopherDB/internal/protocol"
)

var ErrServerAlreadyRunning = errors.New("server is already running")

type Server struct {
	port       string
	dataDir    string
	bufferPool buffer.BufferPoolManager
	catalog    manager.CatalogManager
	sqlService *engine.SqlService

	listener net.Listener
	running  atomic.Bool
	wg       sync.WaitGroup
}

func NewServer(port, dataDir string, bufferPool buffer.BufferPoolManager, catalog manager.CatalogManager, indexManager *index.IndexManager) *Server {
	return &Server{
		port:       port,
		dataDir:    dataDir,
		bufferPool: bufferPool,
		catalog:    catalog,
		sqlService: engine.NewSqlService(dataDir, bufferPool, catalog, indexManager),
	}
}

func (server *Server) Start() error {
	if !server.running.CompareAndSwap(false, true) {
		return ErrServerAlreadyRunning
	}

	listener, err := net.Listen("tcp", ":"+server.port)
	if err != nil {
		server.running.Store(false)
		return err
	}
	server.listener = listener

	slog.Info("server started", "port", server.port, "dataDir", server.dataDir)

	for server.running.Load() {
		connection, err := listener.Accept()
		if err != nil {
			if server.running.Load() {
				slog.Error("accept error", "error", err)
				time.Sleep(50 * time.Millisecond)
			}
			continue
		}

		sessionID := uuid.New().String()
		server.wg.Add(1)
		go server.handleClient(sessionID, connection)
	}

	return nil
}

func (server *Server) Stop() {
	if !server.running.CompareAndSwap(true, false) {
		return
	}

	if server.listener != nil {
		if err := server.listener.Close(); err != nil {
			slog.Warn("listener close error", "error", err)
		}
	}

	server.wg.Wait()
	if err := server.bufferPool.FlushAllPages(); err != nil {
		slog.Warn("flush error", "error", err)
	}
	slog.Info("server stopped")
}

func (server *Server) handleClient(sessionID string, conn net.Conn) {
	defer server.wg.Done()
	defer func() {
		if err := conn.Close(); err != nil {
			slog.Warn("connection close error", "sessionId", sessionID, "error", err)
		}
	}()

	remote := conn.RemoteAddr().String()
	slog.Info("session started", "sessionId", sessionID, "remote", remote)
	defer slog.Info("session ended", "sessionId", sessionID, "remote", remote)

	codec := protocol.NewCodec(conn, conn)

	for {
		request, err := codec.ReadRequest()
		if err != nil {
			if err == io.EOF {
				return
			}
			if isJSONDecodeError(err) {
				response := protocol.NewErrorResponse("", protocol.NewError("SYNTAX", "Invalid JSON request"))
				if writeErr := codec.WriteResponse(response); writeErr != nil {
					slog.Error("session write error", "sessionId", sessionID, "error", writeErr)
					return
				}
				continue
			}
			slog.Error("session read error", "sessionId", sessionID, "error", err)
			return
		}

		response := server.processRequest(sessionID, request)
		if err := codec.WriteResponse(response); err != nil {
			slog.Error("session write error", "sessionId", sessionID, "error", err)
			return
		}
	}
}

func isJSONDecodeError(err error) bool {
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return true
	}
	var typeErr *json.UnmarshalTypeError
	return errors.As(err, &typeErr)
}

func (server *Server) processRequest(sessionID string, request *protocol.Request) *protocol.Response {
	if request == nil {
		return protocol.NewErrorResponse("", protocol.NewError("EXEC", "Request is null"))
	}

	requestID := request.RequestID

	if request.Type != "query" {
		return protocol.NewErrorResponse(requestID, protocol.NewError("EXEC", "Unsupported request type: "+request.Type))
	}

	if request.SQL == "" {
		return protocol.NewErrorResponse(requestID, protocol.NewError("EXEC", "sql is empty"))
	}

	ctx := engine.NewSessionContext(sessionID, requestID, request.Trace)
	result, err := server.sqlService.ExecuteWithContext(ctx, request.SQL)
	if err != nil {
		return protocol.NewErrorResponse(requestID, server.mapExecError(err))
	}

	return protocol.NewOkResponse(requestID, result.Columns(), result.Rows(), result.Affected(), result.Explain())
}

func (server *Server) mapExecError(err error) *protocol.Error {
	var syntaxErr *lexer.SqlSyntaxError
	if errors.As(err, &syntaxErr) {
		return protocol.NewErrorWithPos("SYNTAX", syntaxErr.Message(), syntaxErr.Offset(), syntaxErr.Line(), syntaxErr.Column())
	}

	var semanticErr *semantic.SemanticError
	if errors.As(err, &semanticErr) {
		offset, line, column := semanticErr.Offset(), semanticErr.Line(), semanticErr.Column()
		if offset != nil && line != nil && column != nil {
			return protocol.NewErrorWithPos("SEMANTIC", semanticErr.Message(), *offset, *line, *column)
		}
		return protocol.NewError("SEMANTIC", semanticErr.Message())
	}

	return protocol.NewError("EXEC", err.Error())
}
