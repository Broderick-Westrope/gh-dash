package section

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dlvhdr/gh-dash/v4/internal/config"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/components/prompt"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/context"
)

func TestGetSearchValue_TemplateVars(t *testing.T) {
	tests := []struct {
		name        string
		searchValue string
		checkResult func(t *testing.T, result string)
	}{
		{
			name:        "plain search value passes through unchanged",
			searchValue: "is:open author:@me",
			checkResult: func(t *testing.T, result string) {
				assert.Equal(t, "is:open author:@me", result)
			},
		},
		{
			name:        "nowModify template renders a date string",
			searchValue: `created:>{{ nowModify "-7d" }}`,
			checkResult: func(t *testing.T, result string) {
				assert.False(t, strings.Contains(result, "{{"), "template variables should be rendered, got: %s", result)
				assert.True(t, strings.HasPrefix(result, "created:>"), "prefix should be preserved, got: %s", result)
			},
		},
		{
			name:        "invalid template falls back to raw search value",
			searchValue: "is:open {{ .Unclosed",
			checkResult: func(t *testing.T, result string) {
				assert.Equal(t, "is:open {{ .Unclosed", result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := BaseModel{SearchValue: tt.searchValue}
			result := m.GetSearchValue()
			tt.checkResult(t, result)
		})
	}
}

func TestGetPromptConfirmation(t *testing.T) {
	tests := []struct {
		name         string
		action       string
		view         config.ViewType
		wantNonEmpty bool
	}{
		{
			name:         "done_all in notifications view shows confirmation",
			action:       "done_all",
			view:         config.NotificationsView,
			wantNonEmpty: true,
		},
		{
			name:         "close in PRs view shows confirmation",
			action:       "close",
			view:         config.PRsView,
			wantNonEmpty: true,
		},
		{
			name:         "merge in PRs view shows confirmation",
			action:       "merge",
			view:         config.PRsView,
			wantNonEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &context.ProgramContext{
				View: tt.view,
			}
			m := BaseModel{
				IsPromptConfirmationShown: true,
				PromptConfirmationAction:  tt.action,
				PromptConfirmationBox:     prompt.NewModel(ctx),
			}
			m.Ctx = ctx

			result := m.GetPromptConfirmation()
			if tt.wantNonEmpty {
				require.NotEmpty(t, result, "GetPromptConfirmation() should return non-empty for action %q in view %v", tt.action, tt.view)
			}
		})
	}
}
