package main

import (
	"github.com/charmbracelet/huh"
)

type huhPrompter struct{}

func (this *huhPrompter) Select(prompt string, options []string) (result int, err error) {
	huhOptions := make([]huh.Option[int], len(options))
	for i, opt := range options {
		huhOptions[i] = huh.NewOption(opt, i)
	}
	err = huh.NewSelect[int]().
		Title(prompt).
		Options(huhOptions...).
		Value(&result).
		Run()
	return result, err
}

func (this *huhPrompter) Input(prompt string) (result string, err error) {
	err = huh.NewInput().
		Title(prompt).
		Value(&result).
		Run()
	return result, err
}
