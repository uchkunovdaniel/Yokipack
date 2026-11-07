package main

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// Nozoki's palette. Each colour has a light-terminal and a dark-terminal value,
// because the deep teal that reads well on white is too dark on black.
var (
	teal  = lipgloss.AdaptiveColor{Light: "#1F6F78", Dark: "#5FB3BC"}
	deep  = lipgloss.AdaptiveColor{Light: "#185A62", Dark: "#2C8A94"}
	cream = lipgloss.AdaptiveColor{Light: "#FBF6E9", Dark: "#F3E9D2"}
	ume   = lipgloss.AdaptiveColor{Light: "#C8102E", Dark: "#E5566B"}
	leaf  = lipgloss.AdaptiveColor{Light: "#6E8B3D", Dark: "#9DB86A"}
	muted = lipgloss.AdaptiveColor{Light: "245", Dark: "243"}
	text  = lipgloss.AdaptiveColor{Light: "235", Dark: "252"}
)

// theme is the huh theme for every Yōkipack form.
func theme() *huh.Theme {
	t := huh.ThemeBase()

	f := &t.Focused
	f.Base = f.Base.BorderForeground(deep)
	f.Card = f.Base
	f.Title = f.Title.Foreground(teal).Bold(true)
	f.NoteTitle = f.NoteTitle.Foreground(teal).Bold(true).MarginBottom(1)
	f.Description = f.Description.Foreground(muted)
	f.ErrorIndicator = f.ErrorIndicator.Foreground(ume)
	f.ErrorMessage = f.ErrorMessage.Foreground(ume)
	f.SelectSelector = f.SelectSelector.Foreground(ume)
	f.NextIndicator = f.NextIndicator.Foreground(ume)
	f.PrevIndicator = f.PrevIndicator.Foreground(ume)
	f.Option = f.Option.Foreground(text)
	f.MultiSelectSelector = f.MultiSelectSelector.Foreground(ume)
	f.SelectedOption = f.SelectedOption.Foreground(leaf)
	f.SelectedPrefix = lipgloss.NewStyle().Foreground(leaf).SetString("✓ ")
	f.UnselectedPrefix = lipgloss.NewStyle().Foreground(muted).SetString("• ")
	f.UnselectedOption = f.UnselectedOption.Foreground(text)
	f.FocusedButton = f.FocusedButton.Foreground(cream).Background(teal)
	f.Next = f.FocusedButton
	f.BlurredButton = f.BlurredButton.Foreground(text).Background(lipgloss.AdaptiveColor{Light: "252", Dark: "237"})
	f.TextInput.Cursor = f.TextInput.Cursor.Foreground(leaf)
	f.TextInput.Placeholder = f.TextInput.Placeholder.Foreground(muted)
	f.TextInput.Prompt = f.TextInput.Prompt.Foreground(ume)
	f.TextInput.Text = f.TextInput.Text.Foreground(text)

	// Fields you're not on: same layout, quieter colours.
	t.Blurred = t.Focused
	b := &t.Blurred
	b.Base = b.Base.BorderStyle(lipgloss.HiddenBorder())
	b.Card = b.Base
	b.Title = b.Title.Foreground(muted).Bold(false)
	b.TextInput.Prompt = b.TextInput.Prompt.Foreground(muted)
	b.TextInput.Text = b.TextInput.Text.Foreground(muted)
	b.NextIndicator = lipgloss.NewStyle()
	b.PrevIndicator = lipgloss.NewStyle()

	t.Group.Title = t.Focused.Title
	t.Group.Description = t.Focused.Description
	return t
}
