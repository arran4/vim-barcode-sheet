package main

import (
	"fmt"
	"image/color"
	"log"
	"math"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
	"github.com/fogleman/gg"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
)

// NOTE: The scanner always appends <CR> (Enter). Therefore:
// - All Code values below DO NOT include "<CR>" or a newline.
// - They are mostly ":"-style ex commands where Enter is expected.

// VimOp represents a single barcode entry.
type VimOp struct {
	Code        string // Exact string encoded in the barcode (no <CR>)
	Label       string // Short label shown under barcode
	Description string // Human description
}

// Curated set of multi-keystroke commands where automatic <CR> is useful.
var vimOps = []VimOp{
	// --- Files: write / quit / reload / sudo tricks ---
	{":w", ":w", "Write current file"},
	{":wa", ":wa", "Write all files"},
	{":q", ":q", "Quit (fails if unsaved)"},
	{":wq", ":wq", "Write & quit"},
	{":wqa", ":wqa", "Write & quit all"},
	{":x", ":x", "Write if changed & quit"},
	{":q!", ":q!", "Force quit without saving"},
	{":w!", ":w!", "Force write (read-only files)"},
	{":e!", ":e!", "Reload file (discard changes)"},
	{":up", ":up", "Write only if buffer changed"},
	{":w ++ff=unix", ":w ++ff=unix", "Write with Unix fileformat"},
	{":w ++ff=dos", ":w ++ff=dos", "Write with DOS fileformat"},
	{":!sudo tee %", ":!sudo tee %", "Write as root via sudo tee"},

	// --- Buffer / file navigation ---
	{":ls", ":ls", "List buffers"},
	{":bnext", ":bnext", "Next buffer"},
	{":bprev", ":bprev", "Previous buffer"},
	{":bfirst", ":bfirst", "First buffer"},
	{":blast", ":blast", "Last buffer"},
	{":b#", ":b#", "Alternate buffer"},
	{":bd", ":bd", "Delete current buffer"},
	{":bufdo wqa", ":bufdo wqa", "Write & quit all buffers"},
	{":edit .", ":edit .", "Open file explorer (netrw)"},
	{":Explore", ":Explore", "Netrw file explorer"},
	{":Hexplore", ":Hexplore", "Horizontal explorer split"},
	{":Vexplore", ":Vexplore", "Vertical explorer split"},

	// --- Windows & splits ---
	{":sp", ":sp", "Horizontal split"},
	{":vsp", ":vsp", "Vertical split"},
	{":only", ":only", "Close all other windows"},
	{":close", ":close", "Close current window"},
	{":new", ":new", "New empty window"},
	{":vnew", ":vnew", "New empty vertical split"},
	{":wincmd = ", ":wincmd =", "Equalize split sizes"},
	{":wincmd H", ":wincmd H", "Move window to far left"},
	{":wincmd J", ":wincmd J", "Move window to bottom"},
	{":wincmd K", ":wincmd K", "Move window to top"},
	{":wincmd L", ":wincmd L", "Move window to far right"},

	// --- Tabs ---
	{":tabnew", ":tabnew", "New tab"},
	{":tabclose", ":tabclose", "Close current tab"},
	{":tabonly", ":tabonly", "Close all other tabs"},
	{":tabnext", ":tabnext", "Next tab"},
	{":tabprev", ":tabprev", "Previous tab"},
	{":tabmove 0", ":tabmove 0", "Move tab to front"},
	{":tabmove$", ":tabmove$", "Move tab to end"},

	// --- Search & highlight behaviour ---
	{":noh", ":noh", "Clear search highlight"},
	{":set hlsearch", "hlsearch", "Highlight all search matches"},
	{":set nohlsearch", "nohlsearch", "Disable search highlight"},
	{":set incsearch", "incsearch", "Incremental search"},
	{":set noincsearch", "noincsearch", "Disable incremental search"},
	{":set ignorecase", "ignorecase", "Case-insensitive search"},
	{":set noignorecase", "noignorecase", "Case-sensitive search"},
	{":set smartcase", "smartcase", "Smart case search"},
	{":set nosmartcase", "nosmartcase", "Disable smart case"},

	// --- Indent / tabs / formatting ---
	{":set autoindent", "autoindent", "Enable auto indent"},
	{":set noautoindent", "noautoindent", "Disable auto indent"},
	{":set smartindent", "smartindent", "Enable smart indent"},
	{":set nosmartindent", "nosmartindent", "Disable smart indent"},
	{":set expandtab", "expandtab", "Convert tabs to spaces"},
	{":set noexpandtab", "noexpandtab", "Keep literal tabs"},
	{":set tabstop=2", "ts=2", "Tab width = 2"},
	{":set tabstop=4", "ts=4", "Tab width = 4"},
	{":set shiftwidth=2", "sw=2", "Indent width = 2"},
	{":set shiftwidth=4", "sw=4", "Indent width = 4"},
	{":set softtabstop=2", "sts=2", "Soft tabstop = 2"},
	{":set softtabstop=4", "sts=4", "Soft tabstop = 4"},
	{":retab", ":retab", "Convert indentation to current settings"},

	// --- Background / colours / UI tweaks ---
	{":set background=dark", "bg=dark", "Dark background"},
	{":set background=light", "bg=light", "Light background"},
	{":set number", "number", "Show line numbers"},
	{":set nonumber", "nonumber", "Hide line numbers"},
	{":set relativenumber", "relativenumber", "Relative line numbers"},
	{":set norelativenumber", "norelativenumber", "Disable relative numbers"},
	{":set cursorline", "cursorline", "Highlight current line"},
	{":set nocursorline", "nocursorline", "Disable line highlight"},
	{":set list", "list", "Show invisible chars"},
	{":set nolist", "nolist", "Hide invisible chars"},
	{":set wrap", "wrap", "Wrap long lines"},
	{":set nowrap", "nowrap", "No wrap; horizontal scroll"},
	{":set colorcolumn=80", "cc=80", "Mark column 80"},
	{":set colorcolumn=", "cc=", "Clear colorcolumn"},
	{":set showmatch", "showmatch", "Briefly jump to matching bracket"},
	{":set noshowmatch", "noshowmatch", "Disable showmatch"},
	{":set ruler", "ruler", "Show cursor position"},
	{":set noruler", "noruler", "Hide ruler"},
	{":set showcmd", "showcmd", "Show partial commands"},
	{":set noshowcmd", "noshowcmd", "Hide partial commands"},

	// --- Spellchecking ---
	{":set spell", "spell", "Enable spell checking"},
	{":set nospell", "nospell", "Disable spell checking"},
	{":set spelllang=en_au", "spelllang=en_au", "Set spell lang to en_au"},
	{":set spelllang=en_gb", "spelllang=en_gb", "Set spell lang to en_gb"},

	// --- Mouse / paste / misc convenience ---
	{":set mouse=a", "mouse=a", "Enable mouse in all modes"},
	{":set mouse=", "mouse=", "Disable mouse"},
	{":set paste", "paste", "Enable paste mode"},
	{":set nopaste", "nopaste", "Disable paste mode"},
	{":set clipboard=unnamedplus", "clipboard=unnamedplus", "Use system clipboard"},
	{":set clipboard=", "clipboard=", "Use default Vim registers"},
	{":set foldmethod=indent", "fold=indent", "Fold by indent level"},
	{":set foldmethod=manual", "fold=manual", "Manual folding"},
	{":set foldenable", "foldenable", "Enable folding"},
	{":set nofoldenable", "nofoldenable", "Disable folding"},

	// --- Global substitutions & quick refactors ---
	{":%s/old/new/g", ":%s/old/new/g", "Substitute in whole file"},
	{":%s/old/new/gc", ":%s/old/new/gc", "Substitute with confirm"},
	{":%s/\\s\\+$//e", ":%s/\\s\\+$//e", "Strip trailing whitespace"},
	{":g/DEBUG/d", ":g/DEBUG/d", "Delete all lines containing DEBUG"},
	{":vimgrep /TODO/ **/*", ":vimgrep /TODO/ **/*", "Search TODO in project"},
	{":copen", ":copen", "Open quickfix window"},
	{":cclose", ":cclose", "Close quickfix window"},
}

// font cache so we only parse goregular once per size
var fontCache = map[float64]font.Face{}

func main() {
	// A4 @ 300dpi
	const dpi = 300
	const a4WidthInches = 8.27
	const a4HeightInches = 11.69

	width := int(a4WidthInches * dpi)
	height := int(a4HeightInches * dpi)

	dc := gg.NewContext(width, height)

	// Background
	dc.SetRGB(1, 1, 1)
	dc.Clear()

	margin := 80.0

	// Title using Go Regular
	dc.SetColor(color.Black)
	dc.SetFontFace(mustGoRegularFace(24))
	title := "Vim Barcode Cheat Sheet (Scanner adds <CR>)"
	dc.DrawStringAnchored(title, float64(width)/2, margin/2, 0.5, 0.5)

	// Layout: many commands => 4 columns is a good balance
	cols := 4
	rows := int(math.Ceil(float64(len(vimOps)) / float64(cols)))

	top := margin
	bottom := float64(height) - margin
	left := margin
	right := float64(width) - margin

	cellWidth := (right - left) / float64(cols)
	cellHeight := (bottom - top) / float64(rows)

	barcodeWidth := cellWidth * 0.80
	barcodeHeight := cellHeight * 0.38

	for i, op := range vimOps {
		col := i % cols
		row := i / cols

		x := left + float64(col)*cellWidth
		y := top + float64(row)*cellHeight

		cx := x + cellWidth/2

		// Light cell boundary
		dc.SetLineWidth(0.4)
		dc.SetColor(color.RGBA{R: 230, G: 230, B: 230, A: 255})
		dc.DrawRectangle(x, y, cellWidth, cellHeight)
		dc.Stroke()

		// --- Barcode generation (Code 128) ---
		raw, err := code128.Encode(op.Code) // BarcodeIntCS
		if err != nil {
			log.Printf("encode error for %q: %v", op.Code, err)
			continue
		}

		scaled, err := barcode.Scale(raw, int(barcodeWidth), int(barcodeHeight)) // Barcode
		if err != nil {
			log.Printf("scale error for %q: %v", op.Code, err)
			continue
		}

		// Draw barcode in upper half of the cell
		bx := cx - float64(scaled.Bounds().Dx())/2
		by := y + 6 // top padding inside cell
		dc.DrawImage(scaled, int(bx), int(by))

		// Text under barcode (label + description)
		labelY := by + float64(scaled.Bounds().Dy()) + 8

		dc.SetColor(color.Black)
		dc.SetFontFace(mustGoRegularFace(11))
		dc.DrawStringAnchored(op.Label, cx, labelY, 0.5, 0)

		descY := labelY + 12
		dc.SetFontFace(mustGoRegularFace(8))
		dc.DrawStringWrapped(op.Description, x+6, descY, 0, 0, cellWidth-12, 1.3, gg.AlignCenter)
	}

	out := "vim-barcodes-a4.png"
	if err := dc.SavePNG(out); err != nil {
		log.Fatalf("failed to save PNG: %v", err)
	}

	fmt.Println("Saved:", out)
}

// mustGoRegularFace returns a Go Regular font.Face at the given size.
// Always uses the goregular TTF embedded in the Go font set.
func mustGoRegularFace(size float64) font.Face {
	if face, ok := fontCache[size]; ok {
		return face
	}

	fnt, err := opentype.Parse(goregular.TTF)
	if err != nil {
		log.Fatalf("failed to parse goregular TTF: %v", err)
	}

	face, err := opentype.NewFace(fnt, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatalf("failed to create goregular face (size=%.1f): %v", size, err)
	}

	fontCache[size] = face
	return face
}
