package langserver

import (
	"context"
	"log"
	"os"
	"sync"

	"github.com/hashicorp/terraform/internal/jsonrpc2"
	lsp "github.com/hashicorp/terraform/internal/lsp"
)

// Run starts an LSP server on the supplied stream, and waits until the
// stream is closed.
func Run(ctx context.Context, configPath string, stream jsonrpc2.Stream, opts ...interface{}) error {
	s := &server{
		configPath: configPath,
	}
	conn, client := lsp.RunServer(ctx, stream, s, opts...)
	s.client = client
	return conn.Wait(ctx)
}

type server struct {
	configPath string
	client     lsp.Client

	statusMu sync.Mutex
	active   bool
}

func notImplemented(method string) *jsonrpc2.Error {
	return jsonrpc2.NewErrorf(jsonrpc2.CodeMethodNotFound, "method %q not yet implemented", method)
}

func (s *server) Initialize(context.Context, *lsp.InitializeParams) (*lsp.InitializeResult, error) {
	s.statusMu.Lock()
	defer s.statusMu.Unlock()
	if s.active {
		return nil, jsonrpc2.NewErrorf(jsonrpc2.CodeInvalidRequest, "already initialized")
	}
	s.active = true
	log.Printf("[DEBUG] langserver: Initialize")
	return &lsp.InitializeResult{
		Capabilities: lsp.ServerCapabilities{},
	}, nil
}

func (s *server) Initialized(context.Context, *lsp.InitializedParams) error {
	log.Printf("[DEBUG] langserver: Initialized")
	return nil // nothing to do; this is just a notification from the client
}

func (s *server) Shutdown(context.Context) error {
	s.statusMu.Lock()
	defer s.statusMu.Unlock()
	if !s.active {
		return jsonrpc2.NewErrorf(jsonrpc2.CodeInvalidRequest, "not initialized")
	}
	s.active = false
	return nil
}

func (s *server) Exit(context.Context) error {
	os.Exit(0)
	return nil
}

func (s *server) DidChangeWorkspaceFolders(context.Context, *lsp.DidChangeWorkspaceFoldersParams) error {
	log.Printf("[DEBUG] langserver: DidChangeWorkspaceFolders")
	return notImplemented("DidChangeWorkspaceFolders")
}

func (s *server) DidChangeConfiguration(context.Context, *lsp.DidChangeConfigurationParams) error {
	log.Printf("[DEBUG] langserver: DidChangeConfiguration")
	return notImplemented("DidChangeConfiguration")
}

func (s *server) DidChangeWatchedFiles(context.Context, *lsp.DidChangeWatchedFilesParams) error {
	log.Printf("[DEBUG] langserver: DidChangeWatchedFiles")
	return notImplemented("DidChangeWatchedFiles")
}

func (s *server) Symbols(context.Context, *lsp.WorkspaceSymbolParams) ([]lsp.SymbolInformation, error) {
	log.Printf("[DEBUG] langserver: Symbols")
	return nil, notImplemented("Symbols")
}

func (s *server) ExecuteCommand(context.Context, *lsp.ExecuteCommandParams) (interface{}, error) {
	log.Printf("[DEBUG] langserver: ExecuteCommand")
	return nil, notImplemented("ExecuteCommand")
}

func (s *server) DidOpen(context.Context, *lsp.DidOpenTextDocumentParams) error {
	log.Printf("[DEBUG] langserver: DidOpen")
	return notImplemented("DidOpen")
}

func (s *server) DidChange(context.Context, *lsp.DidChangeTextDocumentParams) error {
	log.Printf("[DEBUG] langserver: DidChange")
	return nil
}

func (s *server) WillSave(context.Context, *lsp.WillSaveTextDocumentParams) error {
	log.Printf("[DEBUG] langserver: WillSave")
	return notImplemented("WillSave")
}

func (s *server) WillSaveWaitUntil(context.Context, *lsp.WillSaveTextDocumentParams) ([]lsp.TextEdit, error) {
	log.Printf("[DEBUG] langserver: WillSaveWaitUntil")
	return nil, notImplemented("WillSaveWaitUntil")
}

func (s *server) DidSave(context.Context, *lsp.DidSaveTextDocumentParams) error {
	log.Printf("[DEBUG] langserver: DidSave")
	return notImplemented("DidSave")
}

func (s *server) DidClose(context.Context, *lsp.DidCloseTextDocumentParams) error {
	log.Printf("[DEBUG] langserver: DidClose")
	return notImplemented("DidClose")
}

func (s *server) Completion(context.Context, *lsp.CompletionParams) (*lsp.CompletionList, error) {
	log.Printf("[DEBUG] langserver: Completion")
	return nil, notImplemented("Completion")
}

func (s *server) CompletionResolve(context.Context, *lsp.CompletionItem) (*lsp.CompletionItem, error) {
	log.Printf("[DEBUG] langserver: CompletionResolve")
	return nil, notImplemented("CompletionResolve")
}

func (s *server) Hover(context.Context, *lsp.TextDocumentPositionParams) (*lsp.Hover, error) {
	log.Printf("[DEBUG] langserver: Hover")
	return nil, notImplemented("Hover")
}

func (s *server) SignatureHelp(context.Context, *lsp.TextDocumentPositionParams) (*lsp.SignatureHelp, error) {
	log.Printf("[DEBUG] langserver: SignatureHelp")
	return nil, notImplemented("SignatureHelp")
}

func (s *server) Definition(context.Context, *lsp.TextDocumentPositionParams) ([]lsp.Location, error) {
	log.Printf("[DEBUG] langserver: Definition")
	return nil, notImplemented("Definition")
}

func (s *server) TypeDefinition(context.Context, *lsp.TextDocumentPositionParams) ([]lsp.Location, error) {
	log.Printf("[DEBUG] langserver: TypeDefinition")
	return nil, notImplemented("TypeDefinition")
}

func (s *server) Implementation(context.Context, *lsp.TextDocumentPositionParams) ([]lsp.Location, error) {
	log.Printf("[DEBUG] langserver: Implementation")
	return nil, notImplemented("Implementation")
}

func (s *server) References(context.Context, *lsp.ReferenceParams) ([]lsp.Location, error) {
	log.Printf("[DEBUG] langserver: References")
	return nil, notImplemented("References")
}

func (s *server) DocumentHighlight(context.Context, *lsp.TextDocumentPositionParams) ([]lsp.DocumentHighlight, error) {
	log.Printf("[DEBUG] langserver: DocumentHighlight")
	return nil, notImplemented("DocumentHighlight")
}

func (s *server) DocumentSymbol(context.Context, *lsp.DocumentSymbolParams) ([]lsp.DocumentSymbol, error) {
	log.Printf("[DEBUG] langserver: DocumentSymbol")
	return nil, notImplemented("DocumentSymbol")
}

func (s *server) CodeAction(context.Context, *lsp.CodeActionParams) ([]lsp.CodeAction, error) {
	log.Printf("[DEBUG] langserver: CodeAction")
	return nil, notImplemented("CodeAction")
}

func (s *server) CodeLens(context.Context, *lsp.CodeLensParams) ([]lsp.CodeLens, error) {
	log.Printf("[DEBUG] langserver: CodeLens")
	return nil, notImplemented("CodeLens")
}

func (s *server) CodeLensResolve(context.Context, *lsp.CodeLens) (*lsp.CodeLens, error) {
	log.Printf("[DEBUG] langserver: CodeLensResolve")
	return nil, notImplemented("CodeLensResolve")
}

func (s *server) DocumentLink(context.Context, *lsp.DocumentLinkParams) ([]lsp.DocumentLink, error) {
	log.Printf("[DEBUG] langserver: DocumentLink")
	return nil, notImplemented("DocumentLink")
}

func (s *server) DocumentLinkResolve(context.Context, *lsp.DocumentLink) (*lsp.DocumentLink, error) {
	log.Printf("[DEBUG] langserver: DocumentLinkResolve")
	return nil, notImplemented("DocumentLinkResolve")
}

func (s *server) DocumentColor(context.Context, *lsp.DocumentColorParams) ([]lsp.ColorInformation, error) {
	log.Printf("[DEBUG] langserver: DocumentColor")
	return nil, notImplemented("DocumentColor")
}

func (s *server) ColorPresentation(context.Context, *lsp.ColorPresentationParams) ([]lsp.ColorPresentation, error) {
	log.Printf("[DEBUG] langserver: ColorPresentation")
	return nil, notImplemented("ColorPresentation")
}

func (s *server) Formatting(context.Context, *lsp.DocumentFormattingParams) ([]lsp.TextEdit, error) {
	log.Printf("[DEBUG] langserver: Formatting")
	return nil, notImplemented("Formatting")
}

func (s *server) RangeFormatting(context.Context, *lsp.DocumentRangeFormattingParams) ([]lsp.TextEdit, error) {
	log.Printf("[DEBUG] langserver: RangeFormatting")
	return nil, notImplemented("RangeFormatting")
}

func (s *server) OnTypeFormatting(context.Context, *lsp.DocumentOnTypeFormattingParams) ([]lsp.TextEdit, error) {
	log.Printf("[DEBUG] langserver: OnTypeFormatting")
	return nil, notImplemented("OnTypeFormatting")
}

func (s *server) Rename(context.Context, *lsp.RenameParams) ([]lsp.WorkspaceEdit, error) {
	log.Printf("[DEBUG] langserver: Rename")
	return nil, notImplemented("Rename")
}

func (s *server) FoldingRanges(context.Context, *lsp.FoldingRangeRequestParam) ([]lsp.FoldingRange, error) {
	log.Printf("[DEBUG] langserver: FoldingRanges")
	return nil, notImplemented("FoldingRanges")
}
