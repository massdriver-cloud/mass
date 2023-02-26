package commands

import tea "github.com/charmbracelet/bubbletea"

// The running tea program.
// Ideally this wouldn't be exposed, but I can't seem to get WithInput to read from the input provided for tests.
// For now this is here to ease testing. One note, passing in an input WithInput does stop tests from being blocked on STDIN
// https://github.com/charmbracelet/bubbletea/discussions/335#discussioncomment-5067445
var P *tea.Program
