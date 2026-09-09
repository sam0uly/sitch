// Package render formats system snapshots as a compact Bento-style dashboard.
package render

import (
	"fmt"
	"image/color"
	"os"
	"slices"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"samouly.fun/sitch/internal/config"
	"samouly.fun/sitch/internal/system"
)

// Print renders info to stdout.
func Print(info system.Info, ascii bool) error {
	return PrintWithOptions(info, Options{Rows: config.Default().Rows, ASCII: ascii})
}

// Options controls the renderer's configured row groups.
type Options struct {
	Rows          [][]string
	ColorMode     string
	Colors        config.CustomColors
	ASCII         bool
	LogoPosition  string // "left", "right", "top", "bottom"
	LogoJustify   string // "top", "middle", "bottom"
	LogoSize      string // "regular" or "small"
	LogoColorMode string // "multi" cycles the theme palette; "single" uses LogoColor
	LogoColor     string // logo_color_mode = "single" accent; empty falls back to the first palette entry
	Logo          string // bundled logo id override; empty means auto-detect
	LogoFile      string // path to a custom ASCII art file; empty means bundled
	Truncate      bool   // if true, clip the logo so it never exceeds the grid height
	FooterAlign   string // "full" or "grid"
}

// PrintWithOptions renders a system snapshot.
func PrintWithOptions(info system.Info, options Options) error {
	setPalette(options.ColorMode, options.Colors)
	if options.Logo != "" {
		if _, ok := lookupLogo(options.Logo, options.LogoSize); !ok {
			preview := LogoNames()
			if len(preview) > 12 {
				preview = preview[:12]
			}
			return fmt.Errorf("unknown logo %q; available ids include: %s", options.Logo, strings.Join(preview, ", "))
		}
	}
	footerMin := lipgloss.Width(renderFooter(info, 0))
	grid, width := renderGridConfigured(facts(info), options.Rows, footerMin)
	footer := renderFooter(info, width)
	for range 4 {
		footerWidth := lipgloss.Width(footer)
		if footerWidth <= width {
			break
		}
		grid, width = renderGridConfigured(facts(info), options.Rows, footerWidth)
		footer = renderFooter(info, width)
	}
	header := renderHeader()
	body := header + "\n" + grid
	logoPosition := options.LogoPosition
	if logoPosition == "" {
		logoPosition = "left"
	}
	logoJustify := options.LogoJustify
	if logoJustify == "" {
		logoJustify = "top"
	}
	footerAlign := options.FooterAlign
	if footerAlign == "" {
		footerAlign = "full"
	}
	logoSize := options.LogoSize
	if logoSize == "" {
		logoSize = "regular"
	}
	setLogoSize(logoSize)
	setLogoColorMode(options.LogoColorMode, options.LogoColor)
	logoText := ""
	if options.LogoFile != "" {
		data, err := os.ReadFile(options.LogoFile)
		if err != nil {
			return fmt.Errorf("read logo file: %w", err)
		}
		if strings.TrimSpace(string(data)) == "" {
			return fmt.Errorf("logo file %q is empty", options.LogoFile)
		}
		logoText = strings.TrimRight(string(data), "\n")
	}
	setLogoRequest(options.Logo, logoText)
	// Truncate caps the logo to the grid's line count. The cap (if any)
	// is applied inside renderLogoBox.
	gridLineCount := strings.Count(grid, "\n") + 1
	if options.Truncate {
		setTruncateHeight(gridLineCount)
	} else {
		setTruncateHeight(0)
	}
	if options.ASCII {
		_, logoWidth := renderLogoBox(info)
		// Re-render the footer to the correct width based on the chosen
		// alignment and the position of the logo so the right edge lines up
		// with the grid's right edge.
		footer = footerForLayout(info, footerAlign, logoPosition, width, logoWidth, options.ASCII)
		body = composeWithLogo(body, footer, width, info, logoWidth, logoPosition, logoJustify, footerAlign)
	} else {
		body = body + "\n" + footer
	}
	composedWidth := max(width, lipgloss.Width(footer), lipgloss.Width(header))
	if options.ASCII {
		_, logoWidth := renderLogoBox(info)
		composedWidth = max(composedWidth, fullComposedWidth(logoPosition, width, logoWidth))
	}
	view := paintSurface(body, composedWidth)
	if _, err := lipgloss.Println(view); err != nil {
		return fmt.Errorf("print fetch: %w", err)
	}
	return nil
}

// fullComposedWidth is the total visible width of a body when the logo is
// shown, given the position. For left/right the body is `width + logoWidth
// + gap` wide; for top/bottom the body is just `width` (the logo sits on
// its own line above or below the grid).
func fullComposedWidth(position string, width, logoWidth int) int {
	gap := 2
	switch position {
	case "top", "bottom":
		return max(width, logoWidth)
	default:
		return width + logoWidth + gap
	}
}

// footerForLayout re-renders the footer at the width that lines up its right
// edge with the grid's right edge in the chosen layout.
func footerForLayout(info system.Info, _ string, logoPosition string, width, logoWidth int, ascii bool) string {
	if !ascii {
		return renderFooter(info, width)
	}
	// For top/bottom layouts the logo and grid share the same horizontal
	// extent, so "full" means max(width, logoWidth) wide. For left/right
	// the footer is always grid-width and is positioned to align with the
	// grid's right edge (the "grid" mode is the only sensible choice).
	switch logoPosition {
	case "top", "bottom":
		return renderFooter(info, max(width, logoWidth))
	default:
		return renderFooter(info, width)
	}
}

// composeWithLogo combines the header + grid body, the ASCII logo, and the
// footer into a single rendered string. The logo can sit on the left,
// right, top or bottom of the grid; in vertical positions (top/bottom)
// the logo is laid out in its own block of lines separated from the grid
// by a blank row so neither overlaps the other.
//
// logoPosition: "left", "right", "top", "bottom" (default: "left")
// logoJustify:  "top", "middle", "bottom" (default: "top") - only used for
//
//	left/right positions; controls vertical alignment of the
//	logo relative to the grid.
//
// footerAlign:  "full" or "grid" (default: "full") - see Options docs.
func composeWithLogo(body, footer string, width int, info system.Info, logoWidth int, logoPosition, logoJustify, footerAlign string) string {
	logo, lw := renderLogoBox(info)
	if logoWidth == 0 {
		logoWidth = lw
	}
	bodyLines := strings.Split(body, "\n")
	headerLine := bodyLines[0]
	// Drop any blank line(s) from the header's bottom margin so the grid
	// sits flush against the logo block.
	gridLines := bodyLines[1:]
	for len(gridLines) > 0 && strings.TrimSpace(ansi.Strip(gridLines[0])) == "" {
		gridLines = gridLines[1:]
	}
	logoLines := strings.Split(logo, "\n")
	gap := 2

	switch logoPosition {
	case "right":
		return composeSide(headerLine, gridLines, footer, width, logoWidth, logoLines, gap, logoJustify, footerAlign, true)
	case "top":
		return composeStacked(headerLine, gridLines, footer, width, logoWidth, logoLines, footerAlign, true)
	case "bottom":
		return composeStacked(headerLine, gridLines, footer, width, logoWidth, logoLines, footerAlign, false)
	default:
		return composeSide(headerLine, gridLines, footer, width, logoWidth, logoLines, gap, logoJustify, footerAlign, false)
	}
}

// composeSide builds the layout where the logo sits beside the grid
// (left or right). The header+grid+footer block stays pinned and contiguous
// so the footer is never cut off; only the logo column gets blank filler
// above/below per logo_justify ("top"/"middle"/"bottom"). Total height is
// max(logo, unit).
func composeSide(headerLine string, gridLines []string, footer string, width, logoWidth int, logoLines []string, gap int, logoJustify, _ string, onRight bool) string {
	paddedLogo := make([]string, len(logoLines))
	for i, l := range logoLines {
		paddedLogo[i] = padRight(l, logoWidth)
	}
	headerRow := padRight(headerLine, width)
	paddedGrid := make([]string, len(gridLines))
	for i, l := range gridLines {
		paddedGrid[i] = padRight(l, width)
	}
	footerLines := strings.Split(footer, "\n")
	unit := make([]string, 0, 1+len(paddedGrid)+len(footerLines))
	unit = append(unit, headerRow)
	unit = append(unit, paddedGrid...)
	unit = append(unit, footerLines...)
	total := max(len(paddedLogo), len(unit))
	unitPadTop := 0
	switch {
	case len(unit) >= total:
		unitPadTop = 0
	case logoJustify == "bottom":
		unitPadTop = total - len(unit)
	case logoJustify == "middle":
		unitPadTop = (total - len(unit)) / 2
	}
	logoPadTop := 0
	switch {
	case len(paddedLogo) >= total:
		logoPadTop = 0
	case logoJustify == "bottom":
		logoPadTop = total - len(paddedLogo)
	case logoJustify == "middle":
		logoPadTop = (total - len(paddedLogo)) / 2
	}
	blankLogo := padRight("", logoWidth)
	blankCell := padRight("", width)
	rows := make([]string, 0, total)
	for row := range total {
		gridCell := blankCell
		unitRow := row - unitPadTop
		if unitRow >= 0 && unitRow < len(unit) {
			gridCell = unit[unitRow]
		}
		logoCell := blankLogo
		logoRow := row - logoPadTop
		if logoRow >= 0 && logoRow < len(paddedLogo) {
			logoCell = paddedLogo[logoRow]
		}
		var line string
		if onRight {
			line = gridCell + strings.Repeat(" ", gap) + logoCell
		} else {
			line = logoCell + strings.Repeat(" ", gap) + gridCell
		}
		rows = append(rows, line)
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// composeStacked builds the layout where the logo sits above (top) or below
// (bottom) the grid in its own block of lines separated by a blank row.
func composeStacked(headerLine string, gridLines []string, footer string, width, logoWidth int, logoLines []string, _ string, onTop bool) string {
	rowWidth := max(width, logoWidth)
	headerRow := padRight(headerLine, rowWidth)
	paddedLogo := make([]string, len(logoLines))
	for i, l := range logoLines {
		paddedLogo[i] = padRight(l, rowWidth)
	}
	paddedGrid := make([]string, len(gridLines))
	for i, l := range gridLines {
		paddedGrid[i] = padRight(l, rowWidth)
	}
	var parts []string
	if onTop {
		// logo | blank row | header + grid
		parts = append(parts, paddedLogo...)
		parts = append(parts, "")
		parts = append(parts, headerRow)
		parts = append(parts, paddedGrid...)
	} else {
		// header + grid | blank row | logo
		parts = append(parts, headerRow)
		parts = append(parts, paddedGrid...)
		parts = append(parts, "")
		parts = append(parts, paddedLogo...)
	}
	composed := lipgloss.JoinVertical(lipgloss.Left, parts...)
	return joinFooter(composed, footer, 0, 0, false, "")
}

// joinFooter attaches the footer underneath the composed body, applying the
// correct horizontal offset so the footer's right edge always lines up
// with the grid's right edge.
func joinFooter(body, footer string, logoWidth, gap int, onRight bool, _ string) string {
	var offset int
	switch {
	case onRight:
		// Logo is on the right: the grid is at the left of the body, so
		// the footer is flush with the grid's left edge.
		offset = 0
	case logoWidth > 0:
		// Logo is on the left: footer is shifted right so it sits under
		// the grid, which starts at `logoWidth + gap`.
		offset = logoWidth + gap
	}
	lines := strings.Split(footer, "\n")
	for i, l := range lines {
		lines[i] = strings.Repeat(" ", offset) + l
	}
	return body + "\n" + strings.Join(lines, "\n")
}

func padRight(value string, width int) string {
	pad := width - lipgloss.Width(value)
	if pad <= 0 {
		return value
	}
	return value + strings.Repeat(" ", pad)
}

func renderGridConfigured(all []fact, configuredRows [][]string, width int) (string, int) {
	byKey := make(map[string]fact, len(all))
	for _, item := range all {
		byKey[item.key] = item
	}
	rows := make([][]gridCell, 0, len(configuredRows))
	colorIndex := 0
	for _, configured := range configuredRows {
		row := make([]gridCell, 0, len(configured))
		for _, key := range configured {
			item, ok := byKey[key]
			if !ok {
				continue
			}
			row = append(row, gridCell{item: item, accent: titleColors[colorIndex%len(titleColors)]})
			colorIndex++
		}
		if len(row) > 0 {
			rows = append(rows, row)
		}
	}
	setTrackWidths(rows, maxColumns(rows), width)
	return renderRows(rows, width)
}

func renderRows(rows [][]gridCell, minimumWidth int) (string, int) {
	rendered, maxWidth := renderRowsOnce(rows)
	if maxWidth < minimumWidth {
		extra := minimumWidth - maxWidth
		for _, row := range rows {
			row[len(row)-1].width += extra
		}
		rendered, maxWidth = renderRowsOnce(rows)
	}
	for i := range rendered {
		if got := lipgloss.Width(rendered[i]); got < maxWidth {
			rendered[i] = lipgloss.PlaceHorizontal(maxWidth, lipgloss.Left, rendered[i])
		}
	}
	return strings.Join(rendered, "\n"), maxWidth
}

func renderRowsOnce(rows [][]gridCell) ([]string, int) {
	rendered := make([]string, len(rows))
	maxWidth := 0
	var above []int
	for i, row := range rows {
		var boundaries []int
		rendered[i], boundaries = renderRow(row, i == 0, i == len(rows)-1, above)
		above = boundaries
		maxWidth = max(maxWidth, lipgloss.Width(rendered[i]))
	}
	return rendered, maxWidth
}

func maxColumns(rows [][]gridCell) int {
	columns := 1
	for _, row := range rows {
		columns = max(columns, len(row))
	}
	return columns
}

func renderHeader() string {
	header := lipgloss.JoinHorizontal(
		lipgloss.Top,
		sitchStyle.Render("Sitch"),
		headerStyle.Render(" • System information"),
	)
	if hasSurfaceBackground {
		return lipgloss.NewStyle().MarginBottom(1).Render(header)
	}
	return header + "\n"
}

func setTrackWidths(rows [][]gridCell, columns, terminalWidth int) {
	_ = columns
	_ = terminalWidth
	columnWidths := make([]int, columns)
	for i := range columnWidths {
		columnWidths[i] = 16
	}
	for _, row := range rows {
		for i, cell := range row {
			columnWidths[i] = max(columnWidths[i], contentWidth(cell.item))
		}
	}
	for _, row := range rows {
		for i := range row {
			row[i].width = columnWidths[i]
		}
		for i := len(row); i < columns; i++ {
			row[len(row)-1].width += columnWidths[i]
		}
	}
}

func contentWidth(item fact) int {
	return max(
		lipgloss.Width(item.icon+" "+item.label),
		ansi.StringWidth(item.value),
	) + 6
}

func renderRow(row []gridCell, top, bottom bool, above []int) (string, []int) {
	height := 2
	for i, cell := range row {
		wrapped := lipgloss.Wrap(cell.item.value, cellContentWidth(cell.width, i == 0, true), " ")
		height = max(height, lipgloss.Height(cell.item.icon+" "+cell.item.label)+lipgloss.Height(wrapped))
	}
	parts := make([]string, len(row))
	for i, cell := range row {
		right := true
		cellTop := top
		parts[i] = renderCell(cell, cellTop, false, i == 0, right, i == len(row)-1, height)
	}
	line := lipgloss.JoinHorizontal(lipgloss.Top, parts...)
	boundaries := make([]int, 0, max(0, len(parts)-1))
	position := 0
	for _, part := range parts[:max(0, len(parts)-1)] {
		position += lipgloss.Width(part)
		boundaries = append(boundaries, position-1)
	}
	if !top {
		line = repairJunctions(line, boundaries, above)
	}
	if bottom {
		line += "\n" + renderBottomEdge(parts, false)
	}
	return line, boundaries
}

func repairJunctions(line string, current, above []int) string {
	positions := make(map[int]string, len(current)+len(above))
	for _, position := range above {
		positions[position] = "┴"
	}
	for _, position := range current {
		if _, continues := positions[position]; continues {
			positions[position] = "┼"
		} else {
			positions[position] = "┬"
		}
	}
	if len(positions) == 0 {
		return line
	}
	ordered := make([]int, 0, len(positions))
	for position := range positions {
		ordered = append(ordered, position)
	}
	slices.Sort(ordered)
	lines := strings.SplitN(line, "\n", 2)
	for _, position := range ordered {
		if position < 0 || position >= ansi.StringWidth(lines[0]) {
			continue
		}
		lines[0] = ansi.Cut(lines[0], 0, position) + borderStyle.Render(positions[position]) + ansi.Cut(lines[0], position+1, ansi.StringWidth(lines[0]))
	}
	if len(lines) == 1 {
		return lines[0]
	}
	return lines[0] + "\n" + lines[1]
}

func renderBottomEdge(parts []string, skipInterior bool) string {
	border := lipgloss.NormalBorder()
	var edge strings.Builder
	for i, part := range parts {
		left := ""
		if i == 0 {
			left = border.BottomLeft
		}
		width := lipgloss.Width(part)
		right := border.MiddleBottom
		if skipInterior && i == len(parts)-2 {
			right = ""
		}
		if i == len(parts)-1 {
			right = border.BottomRight
		}
		inside := max(0, width-lipgloss.Width(left)-lipgloss.Width(right))
		edge.WriteString(borderStyle.Render(left + strings.Repeat(border.Bottom, inside) + right))
	}
	return borderStyle.Render(edge.String())
}

func renderCell(cell gridCell, top, bottom, left, right, last bool, height int) string {
	return renderPanel(cell.item.icon+" "+cell.item.label, cell.accent,
		valueStyle.Render(lipgloss.Wrap(cell.item.value, cellContentWidth(cell.width, left, right), " ")),
		cell.width, max(1, height-1), left, right, bottom, top, last)
}

func cellContentWidth(width int, left, right bool) int {
	edges := 0
	if left {
		edges++
	}
	if right {
		edges++
	}
	return max(1, width-4-edges)
}

func renderPanel(title string, titleColor color.Color, body string, width, height int, left, right, bottom, firstRow, last bool) string {
	border := lipgloss.NormalBorder()
	style := lipgloss.NewStyle().
		Border(border).
		BorderForeground(borderColor).
		BorderTop(false).
		BorderBottom(bottom).
		BorderLeft(left).
		BorderRight(right).
		Padding(0, 2).
		Width(width)
	if !left {
		style = style.PaddingLeft(1).PaddingRight(3)
	}
	bodyLines := strings.Split(fillLines(body, height), "\n")
	bodyRendered := make([]string, len(bodyLines))
	for i, line := range bodyLines {
		bodyRendered[i] = style.Render(line)
	}
	bodyView := strings.Join(bodyRendered, "\n")
	targetWidth := lipgloss.Width(bodyView)
	titleWidth := lipgloss.Width(title)
	leftRune, rightRune := "", ""
	if left {
		if firstRow {
			leftRune = border.TopLeft
		} else {
			leftRune = border.MiddleLeft
		}
	}
	if right {
		if firstRow {
			if last {
				rightRune = border.TopRight
			} else {
				rightRune = border.MiddleTop
			}
		} else {
			if last {
				rightRune = border.MiddleRight
			} else {
				rightRune = border.Middle
			}
		}
	}
	prefix := border.Top + " "
	if left {
		prefix = leftRune + border.Top + " "
	}
	inside := max(0, targetWidth-lipgloss.Width(prefix)-titleWidth-1-lipgloss.Width(rightRune))
	titleView := styleForeground(titleColor).Render(title)
	leftEdge := borderStyle.Render(prefix)
	rightEdge := borderStyle.Render(" " + strings.Repeat(border.Top, inside) + rightRune)
	return leftEdge + titleView + rightEdge + "\n" + bodyView
}

func fillLines(value string, height int) string {
	lines := strings.Split(value, "\n")
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func paintSurface(content string, contentWidth int) string {
	const (
		backgroundPaddingX = 1
		backgroundPaddingY = 1
	)
	width := contentWidth + backgroundPaddingX*2
	linePadding := lipgloss.NewStyle().PaddingLeft(backgroundPaddingX).PaddingRight(backgroundPaddingX)
	lines := strings.Split(content, "\n")
	painted := make([]string, len(lines))
	if !hasSurfaceBackground {
		for i, line := range lines {
			painted[i] = linePadding.Render(line)
		}
		return strings.Join(painted, "\n")
	}
	for i, line := range lines {
		paddedLine := linePadding.Render(line)
		painted[i] = paintLine(lipgloss.PlaceHorizontal(width, lipgloss.Left, paddedLine, lipgloss.WithWhitespaceStyle(surfaceStyle)))
	}
	result := strings.Join(painted, "\n")
	if backgroundPaddingY == 0 {
		return result
	}
	blank := surfaceStyle.Width(width).Render("")
	return strings.Repeat(blank+"\n", backgroundPaddingY) + result + "\n" + strings.Repeat(blank+"\n", backgroundPaddingY-1) + blank
}

func renderFooter(info system.Info, width int) string {
	userContent := valueStyle.Render(info.User + "@" + info.Hostname)
	confetti := mutedStyle.PaddingRight(2).Render("confetti") + colorDots()
	leftWidth := lipgloss.Width(userContent) + 4
	minimumRightWidth := lipgloss.Width(confetti) + 4
	rightWidth := max(width-leftWidth, minimumRightWidth)
	left := footerBody(userContent, borderColor, leftWidth, true, true)
	right := footerBody(confetti, borderColor, rightWidth, false, true)

	border := lipgloss.NormalBorder()
	top := lipgloss.JoinHorizontal(
		lipgloss.Top,
		footerEdge(lipgloss.Width(left), borderColor, border.TopLeft, border.MiddleTop),
		footerEdge(lipgloss.Width(right), borderColor, "", border.TopRight),
	)
	bottom := lipgloss.JoinHorizontal(
		lipgloss.Top,
		footerEdge(lipgloss.Width(left), borderColor, border.BottomLeft, border.MiddleBottom),
		footerEdge(lipgloss.Width(right), borderColor, "", border.BottomRight),
	)
	return top + "\n" + lipgloss.JoinHorizontal(lipgloss.Top, left, right) + "\n" + bottom
}

func footerBody(content string, borderColor color.Color, width int, left, right bool) string {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(borderColor).
		BorderTop(false).
		BorderBottom(false).
		BorderLeft(left).
		BorderRight(right).
		Padding(0, 1).
		Width(width).
		Render(content)
}

func footerEdge(width int, borderColor color.Color, left, right string) string {
	inside := max(0, width-lipgloss.Width(left)-lipgloss.Width(right))
	return lipgloss.NewStyle().Foreground(borderColor).Render(left + strings.Repeat(lipgloss.NormalBorder().Top, inside) + right)
}

func paintLine(line string) string {
	if !hasSurfaceBackground {
		return line
	}
	const reset = "\x1b[m"
	prefix := strings.TrimSuffix(surfaceStyle.Render(" "), " "+reset)
	return surfaceStyle.Render(strings.ReplaceAll(line, reset, reset+prefix)) + reset
}

func colorDots() string {
	const dotCount = 8
	dots := make([]string, dotCount)
	for i, accent := range titleColors[:dotCount] {
		dot := lipgloss.NewStyle().Foreground(accent)
		if i < len(dots)-1 {
			dot = dot.PaddingRight(1)
		}
		dots[i] = dot.Render("●")
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, dots...)
}

func memoryText(info system.Info) string {
	if info.MemoryTotal == 0 {
		return "unknown"
	}
	return strconv.FormatUint(info.MemoryUsed, 10) + " / " + strconv.FormatUint(info.MemoryTotal, 10) + " MiB"
}

func diskText(info system.Info) string {
	if info.DiskTotal == 0 {
		return "unknown"
	}
	return formatBytes(info.DiskUsed) + " / " + formatBytes(info.DiskTotal)
}

func formatBytes(value uint64) string {
	const unit = 1024
	if value < unit {
		return strconv.FormatUint(value, 10) + " B"
	}
	div, exp := uint64(unit), 0
	for n := value / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return strconv.FormatFloat(float64(value)/float64(div), 'f', 1, 64) + " " + string("KMGTPE"[exp]) + "iB"
}
