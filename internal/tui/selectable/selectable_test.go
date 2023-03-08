package selectable_test

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/massdriver-cloud/mass/internal/tui/selectable"
)

func TestSelectedRows(t *testing.T) {
	t.Run("does not render metadata", func(t *testing.T) {
		people := []person{
			{name: "Chauncy", id: "1234567890"},
			{name: "Buddy", id: "987654321"},
		}

		// Focus to make it interactive
		model := newPeopleTable(people).Focused(true)

		got := model.View()

		if strings.Contains(got, "987654321") {
			t.Errorf("did not expect to render metadata\n'%s'", got)
		}
	})

	t.Run("includes metadata", func(t *testing.T) {
		people := []person{
			{name: "Chauncy", id: "1234567890"},
			{name: "Buddy", id: "987654321"},
		}

		// Focus to make it interactive
		model := newPeopleTable(people).Focused(true)

		pressDown := tea.KeyMsg{Type: tea.KeyDown, Alt: false, Runes: []rune{}}
		updatedModel, _ := model.Update(pressDown)

		pressSpace := tea.KeyMsg{Type: tea.KeySpace, Alt: false, Runes: []rune{}}
		updatedModel, _ = updatedModel.Update(pressSpace)

		updatedSelectableModel := (updatedModel).(selectable.Model)
		got := updatedSelectableModel.SelectedRows()
		want := []map[string]interface{}{
			{"name": "Buddy", "id": "987654321"},
		}

		if !reflect.DeepEqual(want, got) {
			t.Errorf("expected %v, got %v", want, got)
		}
	})
}

func TestUsage(t *testing.T) {

	t.Run("'esc' quits; program exits and returns NO selection", func(t *testing.T) {
		people := []person{
			{name: "Chauncy", id: "1234567890"},
			{name: "Buddy", id: "987654321"},
		}

		// Focus to make it interactive
		model := newPeopleTable(people).Focused(true)

		// hover over Buddy
		pressDown := tea.KeyMsg{Type: tea.KeyDown, Alt: false, Runes: []rune{}}
		updatedModel, _ := model.Update(pressDown)

		// select Buddy
		pressSpace := tea.KeyMsg{Type: tea.KeySpace, Alt: false, Runes: []rune{}}
		updatedModel, _ = updatedModel.Update(pressSpace)

		pressEsc := tea.KeyMsg{Type: tea.KeyEsc, Alt: false, Runes: []rune{}}
		updatedModel, _ = updatedModel.Update(pressEsc)

		updatedSelectableModel := (updatedModel).(selectable.Model)
		got := updatedSelectableModel.SelectedRows()

		if len(got) != 0 {
			t.Errorf("expected no results when quitting, got: %v", got)
		}
	})

	t.Run("'s' saves; program exits and returns selection", func(t *testing.T) {
		people := []person{
			{name: "Chauncy", id: "1234567890"},
			{name: "Buddy", id: "987654321"},
		}

		// Focus to make it interactive
		model := newPeopleTable(people).Focused(true)

		pressDown := tea.KeyMsg{Type: tea.KeyDown, Alt: false, Runes: []rune{}}
		updatedModel, _ := model.Update(pressDown)

		pressSpace := tea.KeyMsg{Type: tea.KeySpace, Alt: false, Runes: []rune{}}
		updatedModel, _ = updatedModel.Update(pressSpace)

		pressSave := tea.KeyMsg{Type: tea.KeyRunes, Alt: false, Runes: []rune{'s'}}
		updatedModel, _ = updatedModel.Update(pressSave)

		updatedSelectableModel := (updatedModel).(selectable.Model)
		rows := updatedSelectableModel.SelectedRows()
		got := len(rows)
		want := 1

		if got != want {
			t.Errorf("expected results when saving, got: %d (%v)", got, rows)
		}
	})
}
