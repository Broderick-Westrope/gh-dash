package remindersection

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"

	"github.com/dlvhdr/gh-dash/v4/internal/config"
	"github.com/dlvhdr/gh-dash/v4/internal/data"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/components/prrow"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/components/section"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/components/table"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/constants"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/context"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/keys"
	"github.com/dlvhdr/gh-dash/v4/internal/utils"
)

const SectionType = "reminders"

// ReminderRow wraps standard PR data with the associated reminder entry.
type ReminderRow struct {
	Data  prrow.Data
	Entry data.ReminderEntry
	Key   string
}

// SectionRemindersFetchedMsg is sent when reminder PRs have been fetched.
type SectionRemindersFetchedMsg struct {
	Rows   []ReminderRow
	TaskId string
}

type Model struct {
	section.BaseModel
	Rows []ReminderRow
}

func NewModel(id int, ctx *context.ProgramContext, cfg config.SectionConfig) *Model {
	m := &Model{}
	m.BaseModel = section.NewModel(
		ctx,
		section.NewSectionOptions{
			Id:          id,
			Config:      cfg,
			Type:        SectionType,
			Columns:     getSectionColumns(),
			Singular:    m.GetItemSingularForm(),
			Plural:      m.GetItemPluralForm(),
			LastUpdated: time.Now(),
			CreatedAt:   time.Now(),
		},
	)
	m.Rows = []ReminderRow{}
	return m
}

func getSectionColumns() []table.Column {
	return []table.Column{
		{
			Title: "",
			Width: utils.IntPtr(3),
		},
		{
			Title: "Title",
			Grow:  utils.BoolPtr(true),
		},
		{
			Title: "Note",
			Width: utils.IntPtr(30),
		},
		{
			Title: "Due",
			Width: utils.IntPtr(10),
		},
	}
}

func (m Model) GetItemSingularForm() string {
	return "Reminder"
}

func (m Model) GetItemPluralForm() string {
	return "Reminders"
}

func (m Model) GetTotalCount() int {
	return len(m.Rows)
}

func (m *Model) NumRows() int {
	return len(m.Rows)
}

func (m *Model) GetCurrRow() data.RowData {
	idx := m.Table.GetCurrItem()
	if idx < 0 || idx >= len(m.Rows) {
		return nil
	}
	row := m.Rows[idx]
	return &row.Data
}

func (m Model) GetPagerContent() string {
	pagerContent := ""
	timeElapsed := utils.TimeElapsed(m.LastUpdated())
	if timeElapsed == "now" {
		timeElapsed = "just now"
	} else {
		timeElapsed = fmt.Sprintf("~%v ago", timeElapsed)
	}
	totalCount := len(m.Rows)
	if totalCount > 0 {
		pagerContent = fmt.Sprintf(
			"%v Updated %v • %v %v/%v",
			constants.WaitingIcon,
			timeElapsed,
			m.SingularForm,
			m.Table.GetCurrItem()+1,
			totalCount,
		)
	}
	pager := m.Ctx.Styles.ListViewPort.PagerStyle.Render(pagerContent)
	return pager
}

func (m *Model) SetIsLoading(val bool) {
	m.IsLoading = val
	m.Table.SetIsLoading(val)
}

func (m *Model) ResetRows() {
	m.Rows = nil
	m.BaseModel.ResetRows()
}

func (m *Model) BuildRows() []table.Row {
	var rows []table.Row
	currItem := m.Table.GetCurrItem()
	for i, row := range m.Rows {
		prModel := prrow.PullRequest{
			Ctx:            m.Ctx,
			Data:           &row.Data,
			Columns:        m.Table.Columns,
			ShowAuthorIcon: m.ShowAuthorIcon,
		}

		isSelected := currItem == i
		// Build base state cell from the PR model
		// We build a minimal row: state, title (rendered as extended), note, due
		_ = isSelected
		prTableRow := prModel.ToTableRow(isSelected)

		// Extract state cell (index 0) and title cell.
		// prModel.ToTableRow produces either compact or non-compact format,
		// so we always use index 0 for state and the title column varies.
		// We need just the state icon (col 0) and title (the grow column).
		// Since we define our own 4-column layout, we produce our own row.
		stateCell := prTableRow[0]

		// Render title using the PR's own title render (compact has it at index 2)
		var titleCell string
		if m.Ctx.Config.Theme.Ui.Table.Compact {
			// compact: [state, repo, title, author, assignees, base, comments, review, ci, lines, updated, created]
			if len(prTableRow) > 2 {
				titleCell = prTableRow[2]
			}
		} else {
			// non-compact: [state, extended-title, assignees, base, comments, review, ci, lines, updated, created]
			if len(prTableRow) > 1 {
				titleCell = prTableRow[1]
			}
		}

		noteCell := truncateNote(row.Entry.Note, 28)
		dueCell := formatDueTime(row.Entry.RemindAt)

		rows = append(rows, table.Row{stateCell, titleCell, noteCell, dueCell})
	}

	if rows == nil {
		rows = []table.Row{}
	}

	return rows
}

func (m *Model) dismiss(key string) {
	data.GetRemindersStore().Remove(key)
	filtered := make([]ReminderRow, 0, len(m.Rows))
	for _, row := range m.Rows {
		rowKey := data.ReminderKey(row.Data.Primary.Repository.NameWithOwner, row.Data.Primary.Number)
		if rowKey != key {
			filtered = append(filtered, row)
		}
	}
	m.Rows = filtered
	if m.Ctx != nil {
		m.Table.SetRows(m.BuildRows())
	}
}

func (m *Model) FetchNextPageSectionRows() []tea.Cmd {
	if m == nil {
		return nil
	}

	var cmds []tea.Cmd

	taskId := fmt.Sprintf("fetching_reminders_%d_%s", m.Id, time.Now().String())
	isFirstFetch := m.LastFetchTaskId == ""
	m.LastFetchTaskId = taskId

	task := context.Task{
		Id:           taskId,
		StartText:    fmt.Sprintf(`Fetching reminders for "%s"`, m.Config.Title),
		FinishedText: fmt.Sprintf(`Reminders for "%s" have been fetched`, m.Config.Title),
		State:        context.TaskStart,
		Error:        nil,
	}
	startCmd := m.Ctx.StartTask(task)
	cmds = append(cmds, startCmd)

	// Capture what we need in the closure.
	filters := m.GetFilters()
	sectionId := m.Id
	sectionType := m.Type

	fetchCmd := func() tea.Msg {
		dueEntries := data.GetRemindersStore().GetDue()

		var rows []ReminderRow
		for key, entry := range dueEntries {
			// Parse key format: "owner/repo#number"
			hashIdx := strings.LastIndex(key, "#")
			if hashIdx < 0 {
				log.Warn("reminders: invalid key format, skipping", "key", key)
				data.GetRemindersStore().Remove(key)
				continue
			}
			repoWithOwner := key[:hashIdx]
			numberStr := key[hashIdx+1:]
			number, err := strconv.Atoi(numberStr)
			if err != nil {
				log.Warn("reminders: failed to parse PR number from key", "key", key, "err", err)
				data.GetRemindersStore().Remove(key)
				continue
			}

			// Fetch the PR via search query.
			searchQuery := fmt.Sprintf("repo:%s is:pr %d", repoWithOwner, number)
			res, err := data.FetchPullRequests(searchQuery, 1, nil)
			if err != nil {
				// Transient error (network, rate limit, etc.) — keep the reminder for next tick.
				log.Warn("reminders: failed to fetch PR, will retry", "key", key, "err", err)
				continue
			}
			if len(res.Prs) == 0 {
				// PR is inaccessible or deleted — silently drop the reminder.
				log.Warn("reminders: PR not found, removing reminder", "key", key)
				data.GetRemindersStore().Remove(key)
				continue
			}

			pr := res.Prs[0]

			// Apply client-side filters from m.GetFilters()
			if !matchesFilters(&pr, filters) {
				continue
			}

			rows = append(rows, ReminderRow{
				Data:  prrow.Data{Primary: &pr},
				Entry: entry,
				Key:   key,
			})
		}

		return constants.TaskFinishedMsg{
			SectionId:   sectionId,
			SectionType: sectionType,
			TaskId:      taskId,
			Msg: SectionRemindersFetchedMsg{
				Rows:   rows,
				TaskId: taskId,
			},
		}
	}
	cmds = append(cmds, fetchCmd)

	m.IsLoading = true
	if isFirstFetch {
		m.SetIsLoading(true)
		cmds = append(cmds, m.Table.StartLoadingSpinner())
	}

	return cmds
}

// matchesFilters applies simple client-side filters parsed from the filter string.
// Recognises tokens like "is:open", "is:closed", "is:merged".
// Unrecognised tokens are logged as warnings.
func matchesFilters(pr *data.PullRequestData, filters string) bool {
	if filters == "" {
		return true
	}

	tokens := strings.Fields(filters)
	for _, token := range tokens {
		switch token {
		case "is:open":
			if pr.State != "OPEN" {
				return false
			}
		case "is:closed":
			if pr.State != "CLOSED" {
				return false
			}
		case "is:merged":
			if pr.State != "MERGED" {
				return false
			}
		default:
			log.Warn("reminders: unrecognised filter token, ignoring", "token", token)
		}
	}
	return true
}

func (m *Model) Update(msg tea.Msg) (section.Section, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.IsSearchFocused() {
			switch msg.Type {
			case tea.KeyCtrlC, tea.KeyEsc:
				m.SearchBar.SetValue(m.SearchValue)
				blinkCmd := m.SetIsSearching(false)
				return m, blinkCmd

			case tea.KeyEnter:
				m.SearchValue = m.SearchBar.Value()
				m.SetIsSearching(false)
				m.ResetRows()
				return m, tea.Batch(m.FetchNextPageSectionRows()...)
			}

			break
		}

		switch {
		case key.Matches(msg, keys.PRKeys.DismissReminder):
			idx := m.Table.GetCurrItem()
			if idx >= 0 && idx < len(m.Rows) {
				row := m.Rows[idx]
				m.dismiss(row.Key)
			}
		}

	case SectionRemindersFetchedMsg:
		if m.LastFetchTaskId == msg.TaskId {
			m.Rows = msg.Rows
			m.TotalCount = len(msg.Rows)
			m.SetIsLoading(false)
			m.Table.SetRows(m.BuildRows())
			m.Table.UpdateLastUpdated(time.Now())
			m.UpdateTotalItemsCount(m.TotalCount)
		}
	}

	search, searchCmd := m.SearchBar.Update(msg)
	m.SearchBar = search

	prompt, promptCmd := m.PromptConfirmationBox.Update(msg)
	m.PromptConfirmationBox = prompt

	tableModel, tableCmd := m.Table.Update(msg)
	m.Table = tableModel

	return m, tea.Batch(cmd, searchCmd, promptCmd, tableCmd)
}
