package gui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"strings"
)

const (
	checkboxColumnWidth float32 = 30
	idColumnWidth       float32 = 60
	nameColumnWidth     float32 = 100
	defaultPageSize             = 50
)

// TableConfig holds configuration for table creation
type TableConfig struct {
	Headers           []string
	Data              [][]string
	HasCheckboxes     bool
	OnSelectionChange func(rowIndex int, selected bool, rowData []string)
	OnRowSelected     func(rowIndex int, rowData []string)
	ColumnWidths      map[int]float32
	StatusColumn      int         // -1 if no status column
	TruncateAt        map[int]int // Column index -> max characters before truncation
	ShowTooltips      bool        // Whether to show full text in tooltips
}

// TableFactory creates standardized tables across the application
type TableFactory struct {
	ui *Gui
}

// NewTableFactory creates a new table factory
func NewTableFactory(ui *Gui) *TableFactory {
	return &TableFactory{ui: ui}
}

func (tf *TableFactory) newTableLabel() *widget.Label {
	label := widget.NewLabel("template")
	label.Alignment = fyne.TextAlignLeading
	label.Wrapping = fyne.TextTruncate
	return label
}

// CreateTable creates a standardized table with consistent functionality
func (tf *TableFactory) CreateTable(config TableConfig) *widget.Table {
	if len(config.Data) == 0 {
		// Return empty table with headers
		return tf.createEmptyTable(config.Headers)
	}

	// Track selection state for checkboxes
	selectedRows := make(map[int]bool)

	table := widget.NewTable(
		func() (int, int) {
			return len(config.Data) + 1, len(config.Headers)
		},
		func() fyne.CanvasObject {
			if config.HasCheckboxes {
				return container.NewHBox(
					widget.NewCheck("", nil),
					tf.newTableLabel(),
				)
			}
			return tf.newTableLabel()
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			if config.HasCheckboxes {
				tf.renderCheckboxRow(i, o, config, selectedRows)
			} else {
				tf.renderSimpleRow(i, o, config)
			}
		},
	)

	// Set column widths if specified
	if config.ColumnWidths != nil {
		for col, width := range config.ColumnWidths {
			table.SetColumnWidth(col, width)
		}
	} else {
		// Set default column widths
		tf.setDefaultColumnWidths(table, config)
	}

	// Handle row selection for details view
	table.OnSelected = func(id widget.TableCellID) {
		if id.Row <= 0 || id.Row-1 >= len(config.Data) {
			return
		}

		selectedData := config.Data[id.Row-1]

		if config.OnRowSelected != nil {
			config.OnRowSelected(id.Row-1, selectedData)
		} else {
			tf.showDefaultDetails(config.Headers, selectedData)
		}

		table.UnselectAll()
	}

	return table
}

// renderCheckboxRow renders a row with checkbox functionality
func (tf *TableFactory) renderCheckboxRow(i widget.TableCellID, o fyne.CanvasObject, config TableConfig, selectedRows map[int]bool) {
	container := o.(*fyne.Container)

	if i.Row == 0 {
		// Header row
		if i.Col == 0 {
			// Checkbox column header (empty)
			container.Objects[0].(*widget.Check).Hide()
			container.Objects[1].(*widget.Label).SetText("")
		} else {
			container.Objects[0].(*widget.Check).Hide()
			label := container.Objects[1].(*widget.Label)
			label.SetText(config.Headers[i.Col])
			label.TextStyle = fyne.TextStyle{Bold: true}
		}
	} else {
		// Data rows
		if i.Col == 0 {
			// Checkbox column
			check := container.Objects[0].(*widget.Check)
			check.Show()
			container.Objects[1].(*widget.Label).Hide()

			rowIndex := i.Row - 1
			check.SetChecked(selectedRows[rowIndex])

			check.OnChanged = func(checked bool) {
				selectedRows[rowIndex] = checked
				if config.OnSelectionChange != nil {
					config.OnSelectionChange(rowIndex, checked, config.Data[rowIndex])
				}
			}
		} else {
			container.Objects[0].(*widget.Check).Hide()
			label := container.Objects[1].(*widget.Label)
			label.Show()

			if i.Col < len(config.Data[i.Row-1]) {
				originalText := config.Data[i.Row-1][i.Col]
				displayText := tf.formatCellText(originalText, i.Col, config)
				label.SetText(displayText)
			}
			label.TextStyle = fyne.TextStyle{}

			// Apply status color coding if this is the status column
			tf.applyStatusColor(label, i, config)
		}
	}
}

// renderSimpleRow renders a row without checkboxes
func (tf *TableFactory) renderSimpleRow(i widget.TableCellID, o fyne.CanvasObject, config TableConfig) {
	label := o.(*widget.Label)

	if i.Row == 0 {
		// Header row
		label.SetText(config.Headers[i.Col])
		label.TextStyle = fyne.TextStyle{Bold: true}
	} else {
		// Data row
		if i.Col < len(config.Data[i.Row-1]) {
			originalText := config.Data[i.Row-1][i.Col]
			displayText := tf.formatCellText(originalText, i.Col, config)
			label.SetText(displayText)
		}
		label.TextStyle = fyne.TextStyle{}

		// Apply status color coding if this is the status column
		tf.applyStatusColor(label, i, config)
	}
}

// applyStatusColor applies color coding for status columns
func (tf *TableFactory) applyStatusColor(label *widget.Label, i widget.TableCellID, config TableConfig) {
	if config.StatusColumn >= 0 && i.Col == config.StatusColumn && i.Row > 0 {
		if i.Col < len(config.Data[i.Row-1]) {
			status := strings.ToLower(config.Data[i.Row-1][i.Col])
			switch status {
			case "pending":
				// Default styling
			case "processing", "in_progress":
				// Could add visual indicators here
			case "completed", "success":
				// Could add visual indicators here
			case "failed", "error":
				// Could add visual indicators here
			}
		}
	}
}

// createEmptyTable creates a table with only headers when no data is available
func (tf *TableFactory) createEmptyTable(headers []string) *widget.Table {
	return widget.NewTable(
		func() (int, int) { return 1, len(headers) },
		func() fyne.CanvasObject { return tf.newTableLabel() },
		func(i widget.TableCellID, o fyne.CanvasObject) {
			label := o.(*widget.Label)
			if i.Row == 0 {
				label.SetText(headers[i.Col])
				label.TextStyle = fyne.TextStyle{Bold: true}
			}
		},
	)
}

// setDefaultColumnWidths sets reasonable default column widths with overflow prevention
func (tf *TableFactory) setDefaultColumnWidths(table *widget.Table, config TableConfig) {
	for i, header := range config.Headers {
		var width float32

		switch strings.ToLower(header) {
		case "", "select": // Checkbox column
			width = checkboxColumnWidth
		case "id", "change id":
			width = idColumnWidth
		case "type", "status":
			width = 80
		case "name", "account", "check-in", "entity id":
			width = nameColumnWidth
		case "modified", "created", "date", "timestamp":
			width = 120
		case "description", "details", "message":
			width = 200 // Wider for text content
		case "email", "address":
			width = 150
		default:
			// Calculate width based on content
			width = tf.calculateOptimalWidth(header, i, config)
		}

		table.SetColumnWidth(i, width)
	}
}

// calculateOptimalWidth calculates optimal column width based on header and content
func (tf *TableFactory) calculateOptimalWidth(header string, colIndex int, config TableConfig) float32 {
	// Start with header width
	headerWidth := float32(len(header) * 8) // ~8 pixels per character
	maxContentWidth := headerWidth

	// Sample first few rows to determine content width
	sampleSize := 10
	if len(config.Data) < sampleSize {
		sampleSize = len(config.Data)
	}

	for i := 0; i < sampleSize; i++ {
		if colIndex < len(config.Data[i]) {
			contentLen := len(config.Data[i][colIndex])
			contentWidth := float32(contentLen * 7) // Slightly smaller for content
			if contentWidth > maxContentWidth {
				maxContentWidth = contentWidth
			}
		}
	}

	// Apply reasonable limits
	minWidth := float32(60)
	maxWidth := float32(250)

	if maxContentWidth < minWidth {
		return minWidth
	}
	if maxContentWidth > maxWidth {
		return maxWidth
	}

	return maxContentWidth
}

// truncateText truncates text to specified length and adds ellipsis
func (tf *TableFactory) truncateText(text string, maxLen int) string {
	if maxLen <= 0 || len(text) <= maxLen {
		return text
	}

	if maxLen <= 3 {
		return "..."
	}

	return text[:maxLen-3] + "..."
}

// formatCellText formats cell text with truncation and proper handling
func (tf *TableFactory) formatCellText(text string, colIndex int, config TableConfig) string {
	// Apply truncation if specified for this column
	if config.TruncateAt != nil {
		if maxLen, exists := config.TruncateAt[colIndex]; exists {
			text = tf.truncateText(text, maxLen)
		}
	}

	return text
}

// createEnhancedLabel creates a label with potential tooltip support
func (tf *TableFactory) createEnhancedLabel(text, fullText string, config TableConfig) *widget.Label {
	label := widget.NewLabel(text)

	// Add tooltip if text was truncated and tooltips are enabled
	if config.ShowTooltips && text != fullText && strings.HasSuffix(text, "...") {
		// In Fyne, we can't directly add tooltips to labels, but we can use rich text
		// For now, just return the label - tooltip functionality would need custom widget
	}

	return label
}

// showDefaultDetails shows row details in the right panel
func (tf *TableFactory) showDefaultDetails(headers []string, data []string) {
	var details strings.Builder
	for i, header := range headers {
		if i < len(data) && header != "" {
			details.WriteString(fmt.Sprintf("%s: %s\n", header, data[i]))
		}
	}

	detailsEntry := widget.NewMultiLineEntry()
	detailsEntry.SetText(details.String())
	detailsEntry.Disable()

	tf.ui.ShowDetails(detailsEntry)
}

// CreateSearchableTable creates a table with built-in search functionality
func (tf *TableFactory) CreateSearchableTable(config TableConfig, searchPlaceholder string) fyne.CanvasObject {
	originalData := make([][]string, len(config.Data))
	copy(originalData, config.Data)

	// Create search entry
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder(searchPlaceholder)

	// Create the table with auto-truncation
	table := tf.CreateAutoTruncatedTable(config)

	// Create container to hold filtered table
	tableContainer := container.NewMax(table)

	// Implement search functionality
	searchEntry.OnChanged = func(query string) {
		if query == "" {
			// Show all data
			config.Data = originalData
		} else {
			// Filter data based on search query
			var filteredData [][]string
			query = strings.ToLower(query)

			for _, row := range originalData {
				for _, cell := range row {
					if strings.Contains(strings.ToLower(cell), query) {
						filteredData = append(filteredData, row)
						break
					}
				}
			}
			config.Data = filteredData
		}

		// Recreate table with filtered data
		newTable := tf.CreateAutoTruncatedTable(config)
		tableContainer.Objects = []fyne.CanvasObject{newTable}
		tableContainer.Refresh()
	}

	// Return container with search and table
	return container.NewBorder(
		container.NewPadded(searchEntry),
		nil, nil, nil,
		tableContainer,
	)
}

// CreatePaginatedTable creates a table with pagination controls
func (tf *TableFactory) CreatePaginatedTable(config TableConfig, pageSize int) fyne.CanvasObject {
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}

	originalData := make([][]string, len(config.Data))
	copy(originalData, config.Data)

	currentPage := 0
	totalPages := (len(originalData) + pageSize - 1) / pageSize

	pageLabel := widget.NewLabel(fmt.Sprintf("Page %d of %d", currentPage+1, totalPages))

	// Create the table with auto-truncation
	table := tf.CreateAutoTruncatedTable(config)
	tableContainer := container.NewMax(table)

	var prevBtn, nextBtn *widget.Button

	updateTablePage := func() {
		start := currentPage * pageSize
		end := start + pageSize
		if end > len(originalData) {
			end = len(originalData)
		}

		if start < len(originalData) {
			config.Data = originalData[start:end]
		} else {
			config.Data = [][]string{}
		}

		newTable := tf.CreateAutoTruncatedTable(config)
		tableContainer.Objects = []fyne.CanvasObject{newTable}
		tableContainer.Refresh()

		pageLabel.SetText(fmt.Sprintf("Page %d of %d", currentPage+1, totalPages))
		if currentPage == 0 {
			prevBtn.Disable()
		} else {
			prevBtn.Enable()
		}
		if currentPage >= totalPages-1 {
			nextBtn.Disable()
		} else {
			nextBtn.Enable()
		}
		prevBtn.Refresh()
		nextBtn.Refresh()
	}

	// Create pagination controls
	prevBtn = widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
		if currentPage > 0 {
			currentPage--
			updateTablePage()
		}
	})

	nextBtn = widget.NewButtonWithIcon("", theme.NavigateNextIcon(), func() {
		if currentPage < totalPages-1 {
			currentPage++
			updateTablePage()
		}
	})

	// Initialize first page
	updateTablePage()

	paginationControls := container.NewHBox(
		prevBtn,
		pageLabel,
		nextBtn,
	)

	return container.NewBorder(
		nil,
		container.NewPadded(container.NewCenter(paginationControls)),
		nil, nil,
		tableContainer,
	)
}

// CreateAutoTruncatedTable creates a table with automatic text truncation based on column widths
func (tf *TableFactory) CreateAutoTruncatedTable(config TableConfig) *widget.Table {
	// Set up automatic truncation based on column widths
	if config.TruncateAt == nil {
		config.TruncateAt = make(map[int]int)
	}

	// Calculate truncation lengths based on default column widths
	for i, header := range config.Headers {
		if _, exists := config.TruncateAt[i]; !exists {
			var maxChars int

			switch strings.ToLower(header) {
			case "", "select": // Checkbox column
				maxChars = 0 // No text
			case "id", "change id":
				maxChars = 8
			case "type", "status":
				maxChars = 12
			case "name", "account", "check-in", "entity id":
				maxChars = 15
			case "modified", "created", "date", "timestamp":
				maxChars = 16
			case "description", "details", "message":
				maxChars = 30
			case "email":
				maxChars = 25
			case "address":
				maxChars = 20
			default:
				// Base on calculated width - approximately 7 pixels per character
				maxChars = int(tf.calculateOptimalWidth(header, i, config) / 7)
				if maxChars < 8 {
					maxChars = 8
				}
				if maxChars > 35 {
					maxChars = 35
				}
			}

			if maxChars > 0 {
				config.TruncateAt[i] = maxChars
			}
		}
	}

	// Enable tooltips for truncated content
	config.ShowTooltips = true

	return tf.CreateTable(config)
}

// CreateResponsiveTable creates a table that adapts to container size
func (tf *TableFactory) CreateResponsiveTable(config TableConfig, containerWidth float32) *widget.Table {
	// Adjust column widths based on available space
	totalDefaultWidth := float32(0)
	for i := range config.Headers {
		totalDefaultWidth += tf.calculateOptimalWidth(config.Headers[i], i, config)
	}

	// If total width exceeds container, proportionally reduce
	if totalDefaultWidth > containerWidth && containerWidth > 0 {
		scaleFactor := containerWidth / totalDefaultWidth

		if config.ColumnWidths == nil {
			config.ColumnWidths = make(map[int]float32)
		}

		for i, header := range config.Headers {
			originalWidth := tf.calculateOptimalWidth(header, i, config)
			config.ColumnWidths[i] = originalWidth * scaleFactor
		}
	}

	return tf.CreateAutoTruncatedTable(config)
}
