package langserver

import (
	"context"

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
}

func notImplemented(method string) *jsonrpc2.Error {
	return jsonrpc2.NewErrorf(jsonrpc2.CodeMethodNotFound, "method %q not yet implemented", method)
}

func (s *server) Initialize(context.Context, *lsp.InitializeParams) (*lsp.InitializeResult, error) {
	return nil, notImplemented("Initialize")
}

func (s *server) Initialized(context.Context, *lsp.InitializedParams) error {
	return notImplemented("Initialized")
}

func (s *server) Shutdown(context.Context) error {
	return notImplemented("Shutdown")
}

func (s *server) Exit(context.Context) error {
	return notImplemented("Exit")
}

func (s *server) DidChangeWorkspaceFolders(context.Context, *lsp.DidChangeWorkspaceFoldersParams) error {
	return notImplemented("DidChangeWorkspaceFolders")
}

func (s *server) DidChangeConfiguration(context.Context, *lsp.DidChangeConfigurationParams) error {
	return notImplemented("DidChangeConfiguration")
}

func (s *server) DidChangeWatchedFiles(context.Context, *lsp.DidChangeWatchedFilesParams) error {
	return notImplemented("DidChangeWatchedFiles")
}

func (s *server) Symbols(context.Context, *lsp.WorkspaceSymbolParams) ([]lsp.SymbolInformation, error) {
	return nil, notImplemented("Symbols")
}

func (s *server) ExecuteCommand(context.Context, *lsp.ExecuteCommandParams) (interface{}, error) {
	return nil, notImplemented("ExecuteCommand")
}

func (s *server) DidOpen(context.Context, *lsp.DidOpenTextDocumentParams) error {
	return notImplemented("DidOpen")
}

func (s *server) DidChange(context.Context, *lsp.DidChangeTextDocumentParams) error {
	return nil
}

func (s *server) WillSave(context.Context, *lsp.WillSaveTextDocumentParams) error {
	return notImplemented("WillSave")
}

func (s *server) WillSaveWaitUntil(context.Context, *lsp.WillSaveTextDocumentParams) ([]lsp.TextEdit, error) {
	return nil, notImplemented("WillSaveWaitUntil")
}

func (s *server) DidSave(context.Context, *lsp.DidSaveTextDocumentParams) error {
	return notImplemented("DidSave")
}

func (s *server) DidClose(context.Context, *lsp.DidCloseTextDocumentParams) error {
	return notImplemented("DidClose")
}

func (s *server) Completion(context.Context, *lsp.CompletionParams) (*lsp.CompletionList, error) {
	return nil, notImplemented("Completion")
}

func (s *server) CompletionResolve(context.Context, *lsp.CompletionItem) (*lsp.CompletionItem, error) {
	return nil, notImplemented("CompletionResolve")
}

func (s *server) Hover(context.Context, *lsp.TextDocumentPositionParams) (*lsp.Hover, error) {
	return nil, notImplemented("Hover")
}

func (s *server) SignatureHelp(context.Context, *lsp.TextDocumentPositionParams) (*lsp.SignatureHelp, error) {
	return nil, notImplemented("SignatureHelp")
}

func (s *server) Definition(context.Context, *lsp.TextDocumentPositionParams) ([]lsp.Location, error) {
	return nil, notImplemented("Definition")
}

func (s *server) TypeDefinition(context.Context, *lsp.TextDocumentPositionParams) ([]lsp.Location, error) {
	return nil, notImplemented("TypeDefinition")
}

func (s *server) Implementation(context.Context, *lsp.TextDocumentPositionParams) ([]lsp.Location, error) {
	return nil, notImplemented("Implementation")
}

func (s *server) References(context.Context, *lsp.ReferenceParams) ([]lsp.Location, error) {
	return nil, notImplemented("References")
}

func (s *server) DocumentHighlight(context.Context, *lsp.TextDocumentPositionParams) ([]lsp.DocumentHighlight, error) {
	return nil, notImplemented("DocumentHighlight")
}

func (s *server) DocumentSymbol(context.Context, *lsp.DocumentSymbolParams) ([]lsp.DocumentSymbol, error) {
	return nil, notImplemented("DocumentSymbol")
}

func (s *server) CodeAction(context.Context, *lsp.CodeActionParams) ([]lsp.CodeAction, error) {
	return nil, notImplemented("CodeAction")
}

func (s *server) CodeLens(context.Context, *lsp.CodeLensParams) ([]lsp.CodeLens, error) {
	return nil, notImplemented("CodeLens")
}

func (s *server) CodeLensResolve(context.Context, *lsp.CodeLens) (*lsp.CodeLens, error) {
	return nil, notImplemented("CodeLensResolve")
}

func (s *server) DocumentLink(context.Context, *lsp.DocumentLinkParams) ([]lsp.DocumentLink, error) {
	return nil, notImplemented("DocumentLink")
}

func (s *server) DocumentLinkResolve(context.Context, *lsp.DocumentLink) (*lsp.DocumentLink, error) {
	return nil, notImplemented("DocumentLinkResolve")
}

func (s *server) DocumentColor(context.Context, *lsp.DocumentColorParams) ([]lsp.ColorInformation, error) {
	return nil, notImplemented("DocumentColor")
}

func (s *server) ColorPresentation(context.Context, *lsp.ColorPresentationParams) ([]lsp.ColorPresentation, error) {
	return nil, notImplemented("ColorPresentation")
}

func (s *server) Formatting(context.Context, *lsp.DocumentFormattingParams) ([]lsp.TextEdit, error) {
	return nil, notImplemented("Formatting")
}

func (s *server) RangeFormatting(context.Context, *lsp.DocumentRangeFormattingParams) ([]lsp.TextEdit, error) {
	return nil, notImplemented("RangeFormatting")
}

func (s *server) OnTypeFormatting(context.Context, *lsp.DocumentOnTypeFormattingParams) ([]lsp.TextEdit, error) {
	return nil, notImplemented("OnTypeFormatting")
}

func (s *server) Rename(context.Context, *lsp.RenameParams) ([]lsp.WorkspaceEdit, error) {
	return nil, notImplemented("Rename")
}

func (s *server) FoldingRanges(context.Context, *lsp.FoldingRangeRequestParam) ([]lsp.FoldingRange, error) {
	return nil, notImplemented("FoldingRanges")
}
