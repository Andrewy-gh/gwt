package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Andrewy-gh/gwt/internal/config"
	"github.com/Andrewy-gh/gwt/internal/tui/styles"
)

// ConfigField represents a configuration field with its metadata
type ConfigField struct {
	Name        string      // Display name
	Key         string      // Config key path (e.g., "docker.port_offset")
	Type        FieldType   // Field type (string, bool, int, array)
	Value       interface{} // Current value
	Description string      // Help text
	Section     string      // Section name for grouping
}

// FieldType represents the type of a configuration field
type FieldType int

const (
	FieldTypeString FieldType = iota
	FieldTypeBool
	FieldTypeInt
	FieldTypeStringArray
)

// ConfigEditorModel is the configuration editor view
type ConfigEditorModel struct {
	repoPath    string
	cfg         *config.Config
	fields      []ConfigField
	cursor      int
	offset      int
	width       int
	height      int
	editing     bool
	editInput   textinput.Model
	editIndex   int
	arrayMode   bool     // In array editing mode
	arrayItems  []string // Items in current array
	arrayCursor int      // Cursor within array
	cancelled   bool
	saved       bool
	dirty       bool // Config has unsaved changes
	err         error
}

// NewConfigEditorModel creates a new config editor view
func NewConfigEditorModel(repoPath string) *ConfigEditorModel {
	ti := textinput.New()
	ti.CharLimit = 200
	ti.Width = 50

	return &ConfigEditorModel{
		repoPath:  repoPath,
		editInput: ti,
	}
}

// Init initializes the config editor
func (m *ConfigEditorModel) Init() tea.Cmd {
	return m.loadConfig()
}

// configLoadedMsg is sent when config is loaded
type configLoadedMsg struct {
	cfg *config.Config
	err error
}

// loadConfig loads the configuration
func (m *ConfigEditorModel) loadConfig() tea.Cmd {
	return func() tea.Msg {
		cfg, err := config.Load(m.repoPath)
		return configLoadedMsg{cfg: cfg, err: err}
	}
}

// buildFields builds the list of editable fields from config
func (m *ConfigEditorModel) buildFields() {
	m.fields = []ConfigField{
		// Copy settings
		{
			Name:        "Copy Defaults",
			Key:         "copy_defaults",
			Type:        FieldTypeStringArray,
			Value:       m.cfg.CopyDefaults,
			Description: "Default files/dirs to copy to new worktrees",
			Section:     "Copy Settings",
		},
		{
			Name:        "Copy Exclude",
			Key:         "copy_exclude",
			Type:        FieldTypeStringArray,
			Value:       m.cfg.CopyExclude,
			Description: "Files/dirs to exclude from copying",
			Section:     "Copy Settings",
		},
		// Docker settings
		{
			Name:        "Compose Files",
			Key:         "docker.compose_files",
			Type:        FieldTypeStringArray,
			Value:       m.cfg.Docker.ComposeFiles,
			Description: "Docker Compose file paths",
			Section:     "Docker",
		},
		{
			Name:        "Data Directories",
			Key:         "docker.data_directories",
			Type:        FieldTypeStringArray,
			Value:       m.cfg.Docker.DataDirectories,
			Description: "Docker data directories to symlink",
			Section:     "Docker",
		},
		{
			Name:        "Default Mode",
			Key:         "docker.default_mode",
			Type:        FieldTypeString,
			Value:       m.cfg.Docker.DefaultMode,
			Description: "Default Docker mode (shared/new)",
			Section:     "Docker",
		},
		{
			Name:        "Port Offset",
			Key:         "docker.port_offset",
			Type:        FieldTypeInt,
			Value:       m.cfg.Docker.PortOffset,
			Description: "Port offset for Docker services",
			Section:     "Docker",
		},
		// Dependencies settings
		{
			Name:        "Auto Install",
			Key:         "dependencies.auto_install",
			Type:        FieldTypeBool,
			Value:       m.cfg.Dependencies.AutoInstall,
			Description: "Auto-install dependencies on create",
			Section:     "Dependencies",
		},
		{
			Name:        "Paths",
			Key:         "dependencies.paths",
			Type:        FieldTypeStringArray,
			Value:       m.cfg.Dependencies.Paths,
			Description: "Paths to check for dependencies",
			Section:     "Dependencies",
		},
		// Migrations settings
		{
			Name:        "Auto Detect",
			Key:         "migrations.auto_detect",
			Type:        FieldTypeBool,
			Value:       m.cfg.Migrations.AutoDetect,
			Description: "Auto-detect migration framework",
			Section:     "Migrations",
		},
		{
			Name:        "Command",
			Key:         "migrations.command",
			Type:        FieldTypeString,
			Value:       m.cfg.Migrations.Command,
			Description: "Migration command to run",
			Section:     "Migrations",
		},
		// Hooks settings
		{
			Name:        "Post Create Hooks",
			Key:         "hooks.post_create",
			Type:        FieldTypeStringArray,
			Value:       m.cfg.Hooks.PostCreate,
			Description: "Commands to run after worktree creation",
			Section:     "Hooks",
		},
		{
			Name:        "Post Delete Hooks",
			Key:         "hooks.post_delete",
			Type:        FieldTypeStringArray,
			Value:       m.cfg.Hooks.PostDelete,
			Description: "Commands to run after worktree deletion",
			Section:     "Hooks",
		},
	}
}

// Update handles messages
func (m *ConfigEditorModel) Update(msg tea.Msg) (*ConfigEditorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case configLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.cfg = msg.cfg
			m.buildFields()
		}

	case tea.KeyMsg:
		if m.err != nil {
			return m, nil
		}

		// Handle array editing mode
		if m.arrayMode {
			return m.handleArrayInput(msg)
		}

		// Handle field editing mode
		if m.editing {
			return m.handleEditInput(msg)
		}

		// Normal navigation mode
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			m.cursorUp()
		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			m.cursorDown()
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter", " "))):
			m.startEditing()
		case key.Matches(msg, key.NewBinding(key.WithKeys("s"))):
			if m.dirty {
				return m, m.saveConfig()
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
			// Reload config
			return m, m.loadConfig()
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc", "q"))):
			m.cancelled = true
		}
	}

	return m, nil
}

// handleEditInput handles input during field editing
func (m *ConfigEditorModel) handleEditInput(msg tea.KeyMsg) (*ConfigEditorModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.editing = false
		m.editInput.Blur()
		return m, nil
	case "enter":
		m.applyEdit()
		m.editing = false
		m.editInput.Blur()
		return m, nil
	default:
		var cmd tea.Cmd
		m.editInput, cmd = m.editInput.Update(msg)
		return m, cmd
	}
}

// handleArrayInput handles input during array editing
func (m *ConfigEditorModel) handleArrayInput(msg tea.KeyMsg) (*ConfigEditorModel, tea.Cmd) {
	if m.editing {
		// Editing an array item
		switch msg.String() {
		case "esc":
			m.editing = false
			m.editInput.Blur()
			return m, nil
		case "enter":
			// Apply the edit to array item
			if m.editIndex < len(m.arrayItems) {
				m.arrayItems[m.editIndex] = m.editInput.Value()
			} else {
				// Adding new item
				m.arrayItems = append(m.arrayItems, m.editInput.Value())
			}
			m.editing = false
			m.editInput.Blur()
			m.dirty = true
			return m, nil
		default:
			var cmd tea.Cmd
			m.editInput, cmd = m.editInput.Update(msg)
			return m, cmd
		}
	}

	// Array navigation mode
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		if m.arrayCursor > 0 {
			m.arrayCursor--
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
		if m.arrayCursor < len(m.arrayItems) {
			m.arrayCursor++
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter", " "))):
		// Edit selected item or add new
		m.editing = true
		m.editIndex = m.arrayCursor
		if m.arrayCursor < len(m.arrayItems) {
			m.editInput.SetValue(m.arrayItems[m.arrayCursor])
		} else {
			m.editInput.SetValue("")
			m.editInput.Placeholder = "Enter new item..."
		}
		m.editInput.Focus()
		return m, textinput.Blink
	case key.Matches(msg, key.NewBinding(key.WithKeys("d", "delete"))):
		// Delete selected item
		if m.arrayCursor < len(m.arrayItems) {
			m.arrayItems = append(m.arrayItems[:m.arrayCursor], m.arrayItems[m.arrayCursor+1:]...)
			if m.arrayCursor >= len(m.arrayItems) && m.arrayCursor > 0 {
				m.arrayCursor--
			}
			m.dirty = true
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("a"))):
		// Add new item at end
		m.arrayCursor = len(m.arrayItems)
		m.editing = true
		m.editIndex = m.arrayCursor
		m.editInput.SetValue("")
		m.editInput.Placeholder = "Enter new item..."
		m.editInput.Focus()
		return m, textinput.Blink
	case key.Matches(msg, key.NewBinding(key.WithKeys("esc", "q"))):
		// Exit array mode and save changes
		m.saveArrayToField()
		m.arrayMode = false
	}

	return m, nil
}

// cursorUp moves the cursor up
func (m *ConfigEditorModel) cursorUp() {
	if m.cursor > 0 {
		m.cursor--
		if m.cursor < m.offset {
			m.offset = m.cursor
		}
	}
}

// cursorDown moves the cursor down
func (m *ConfigEditorModel) cursorDown() {
	if m.cursor < len(m.fields)-1 {
		m.cursor++
		maxVisible := m.height - 12
		if maxVisible < 5 {
			maxVisible = 5
		}
		if m.cursor >= m.offset+maxVisible {
			m.offset = m.cursor - maxVisible + 1
		}
	}
}

// startEditing starts editing the current field
func (m *ConfigEditorModel) startEditing() {
	if m.cursor >= len(m.fields) {
		return
	}

	field := &m.fields[m.cursor]
	m.editIndex = m.cursor

	switch field.Type {
	case FieldTypeBool:
		// Toggle boolean directly
		if v, ok := field.Value.(bool); ok {
			field.Value = !v
			m.updateConfigField(field)
			m.dirty = true
		}
	case FieldTypeStringArray:
		// Enter array editing mode
		m.arrayMode = true
		m.arrayCursor = 0
		if arr, ok := field.Value.([]string); ok {
			m.arrayItems = make([]string, len(arr))
			copy(m.arrayItems, arr)
		} else {
			m.arrayItems = []string{}
		}
	default:
		// String or Int - use text input
		m.editing = true
		switch v := field.Value.(type) {
		case string:
			m.editInput.SetValue(v)
		case int:
			m.editInput.SetValue(fmt.Sprintf("%d", v))
		default:
			m.editInput.SetValue(fmt.Sprintf("%v", field.Value))
		}
		m.editInput.Focus()
	}
}

// applyEdit applies the current edit to the field
func (m *ConfigEditorModel) applyEdit() {
	if m.editIndex >= len(m.fields) {
		return
	}

	field := &m.fields[m.editIndex]
	value := m.editInput.Value()

	switch field.Type {
	case FieldTypeString:
		field.Value = value
	case FieldTypeInt:
		var intVal int
		fmt.Sscanf(value, "%d", &intVal)
		field.Value = intVal
	}

	m.updateConfigField(field)
	m.dirty = true
}

// saveArrayToField saves the current array items to the field
func (m *ConfigEditorModel) saveArrayToField() {
	if m.editIndex >= len(m.fields) {
		return
	}

	field := &m.fields[m.editIndex]
	// Filter out empty items
	filtered := make([]string, 0, len(m.arrayItems))
	for _, item := range m.arrayItems {
		if strings.TrimSpace(item) != "" {
			filtered = append(filtered, item)
		}
	}
	field.Value = filtered
	m.updateConfigField(field)
}

// updateConfigField updates the config struct with the field value
func (m *ConfigEditorModel) updateConfigField(field *ConfigField) {
	switch field.Key {
	case "copy_defaults":
		m.cfg.CopyDefaults = field.Value.([]string)
	case "copy_exclude":
		m.cfg.CopyExclude = field.Value.([]string)
	case "docker.compose_files":
		m.cfg.Docker.ComposeFiles = field.Value.([]string)
	case "docker.data_directories":
		m.cfg.Docker.DataDirectories = field.Value.([]string)
	case "docker.default_mode":
		m.cfg.Docker.DefaultMode = field.Value.(string)
	case "docker.port_offset":
		m.cfg.Docker.PortOffset = field.Value.(int)
	case "dependencies.auto_install":
		m.cfg.Dependencies.AutoInstall = field.Value.(bool)
	case "dependencies.paths":
		m.cfg.Dependencies.Paths = field.Value.([]string)
	case "migrations.auto_detect":
		m.cfg.Migrations.AutoDetect = field.Value.(bool)
	case "migrations.command":
		m.cfg.Migrations.Command = field.Value.(string)
	case "hooks.post_create":
		m.cfg.Hooks.PostCreate = field.Value.([]string)
	case "hooks.post_delete":
		m.cfg.Hooks.PostDelete = field.Value.([]string)
	}
}

// configSavedMsg is sent when config is saved
type configSavedMsg struct {
	err error
}

// saveConfig saves the configuration to disk
func (m *ConfigEditorModel) saveConfig() tea.Cmd {
	return func() tea.Msg {
		err := config.Save(m.repoPath, m.cfg)
		return configSavedMsg{err: err}
	}
}

// View renders the view
func (m *ConfigEditorModel) View(width, height int) string {
	m.width = width
	m.height = height

	if m.err != nil {
		return m.renderError()
	}

	if m.cfg == nil {
		return m.renderLoading()
	}

	if m.arrayMode {
		return m.renderArrayEditor()
	}

	return m.renderFieldList()
}

// renderLoading renders the loading state
func (m *ConfigEditorModel) renderLoading() string {
	var b strings.Builder
	b.WriteString(styles.Title.Render("Configuration Editor"))
	b.WriteString("\n\n")
	b.WriteString(styles.MutedText.Render("Loading configuration..."))
	b.WriteString("\n")
	return b.String()
}

// renderError renders an error message
func (m *ConfigEditorModel) renderError() string {
	var b strings.Builder
	b.WriteString(styles.Title.Render("Configuration Editor"))
	b.WriteString("\n\n")
	b.WriteString(styles.ErrorText.Render("Error: " + m.err.Error()))
	b.WriteString("\n\n")
	b.WriteString(styles.MutedText.Render("Press esc to return to menu"))
	b.WriteString("\n")
	return b.String()
}

// renderFieldList renders the field list view
func (m *ConfigEditorModel) renderFieldList() string {
	var b strings.Builder

	// Title
	title := "Configuration Editor"
	if m.dirty {
		title += " (modified)"
	}
	b.WriteString(styles.Title.Render(title))
	b.WriteString("\n\n")

	// Edit input (if editing)
	if m.editing {
		b.WriteString(styles.Selected.Render("Editing: "))
		b.WriteString(m.fields[m.editIndex].Name)
		b.WriteString("\n")
		b.WriteString(m.editInput.View())
		b.WriteString("\n")
		b.WriteString(styles.MutedText.Render("Enter: save, Esc: cancel"))
		b.WriteString("\n\n")
	}

	// Field list
	maxVisible := m.height - 12
	if maxVisible < 5 {
		maxVisible = 5
	}

	visibleEnd := m.offset + maxVisible
	if visibleEnd > len(m.fields) {
		visibleEnd = len(m.fields)
	}

	currentSection := ""
	for i := m.offset; i < visibleEnd; i++ {
		field := m.fields[i]

		// Section header
		if field.Section != currentSection {
			currentSection = field.Section
			b.WriteString("\n")
			b.WriteString(styles.Subtitle.Render("─── " + currentSection + " ───"))
			b.WriteString("\n")
		}

		// Field row
		cursor := "  "
		if i == m.cursor {
			cursor = styles.Cursor.Render("▸ ")
		}

		name := field.Name
		if i == m.cursor {
			name = lipgloss.NewStyle().Foreground(styles.Primary).Bold(true).Render(name)
		}

		value := m.formatFieldValue(field)

		b.WriteString(fmt.Sprintf("%s%-22s %s\n", cursor, name, value))
	}

	// Scroll indicators
	if m.offset > 0 {
		b.WriteString(styles.MutedText.Render("  ▲ More above"))
		b.WriteString("\n")
	}
	if visibleEnd < len(m.fields) {
		b.WriteString(styles.MutedText.Render("  ▼ More below"))
		b.WriteString("\n")
	}

	// Help
	b.WriteString("\n")
	b.WriteString(styles.Help.Render("↑/k, ↓/j: navigate  Enter: edit  S: save  R: reload  Esc: back"))
	b.WriteString("\n")

	return b.String()
}

// renderArrayEditor renders the array editing view
func (m *ConfigEditorModel) renderArrayEditor() string {
	var b strings.Builder

	field := m.fields[m.editIndex]
	b.WriteString(styles.Title.Render("Edit: " + field.Name))
	b.WriteString("\n")
	b.WriteString(styles.MutedText.Render(field.Description))
	b.WriteString("\n\n")

	// Edit input (if editing an item)
	if m.editing {
		if m.editIndex < len(m.arrayItems) {
			b.WriteString(styles.Selected.Render("Editing item:"))
		} else {
			b.WriteString(styles.Selected.Render("Add new item:"))
		}
		b.WriteString("\n")
		b.WriteString(m.editInput.View())
		b.WriteString("\n")
		b.WriteString(styles.MutedText.Render("Enter: save, Esc: cancel"))
		b.WriteString("\n\n")
	}

	// Array items
	if len(m.arrayItems) == 0 {
		b.WriteString(styles.MutedText.Render("  (no items)"))
		b.WriteString("\n")
	} else {
		for i, item := range m.arrayItems {
			cursor := "  "
			if i == m.arrayCursor && !m.editing {
				cursor = styles.Cursor.Render("▸ ")
			}

			itemText := item
			if i == m.arrayCursor && !m.editing {
				itemText = lipgloss.NewStyle().Foreground(styles.Primary).Render(item)
			}

			b.WriteString(fmt.Sprintf("%s%s\n", cursor, itemText))
		}
	}

	// "Add new" option
	addCursor := "  "
	if m.arrayCursor == len(m.arrayItems) && !m.editing {
		addCursor = styles.Cursor.Render("▸ ")
	}
	addText := "(add new item)"
	if m.arrayCursor == len(m.arrayItems) && !m.editing {
		addText = lipgloss.NewStyle().Foreground(styles.Primary).Render(addText)
	}
	b.WriteString(fmt.Sprintf("\n%s%s\n", addCursor, addText))

	// Help
	b.WriteString("\n")
	b.WriteString(styles.Help.Render("↑/k, ↓/j: navigate  Enter: edit  A: add  D: delete  Esc: done"))
	b.WriteString("\n")

	return b.String()
}

// formatFieldValue formats a field value for display
func (m *ConfigEditorModel) formatFieldValue(field ConfigField) string {
	switch field.Type {
	case FieldTypeBool:
		if v, ok := field.Value.(bool); ok {
			if v {
				return styles.SuccessText.Render("✓ true")
			}
			return styles.MutedText.Render("✗ false")
		}
	case FieldTypeInt:
		if v, ok := field.Value.(int); ok {
			return styles.MutedText.Render(fmt.Sprintf("%d", v))
		}
	case FieldTypeString:
		if v, ok := field.Value.(string); ok {
			if v == "" {
				return styles.MutedText.Render("(empty)")
			}
			if len(v) > 30 {
				return styles.MutedText.Render(v[:27] + "...")
			}
			return styles.MutedText.Render(v)
		}
	case FieldTypeStringArray:
		if arr, ok := field.Value.([]string); ok {
			if len(arr) == 0 {
				return styles.MutedText.Render("(empty)")
			}
			return styles.MutedText.Render(fmt.Sprintf("[%d items]", len(arr)))
		}
	}
	return styles.MutedText.Render("(unknown)")
}

// IsCancelled returns true if the user cancelled
func (m *ConfigEditorModel) IsCancelled() bool {
	return m.cancelled
}

// IsSaved returns true if the config was saved
func (m *ConfigEditorModel) IsSaved() bool {
	return m.saved
}
