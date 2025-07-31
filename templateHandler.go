package main

import (
	"html/template"
)

func ExecuteHtmlFile(file string) (*template.Template, error) {
	t, err := template.ParseFiles(file)
	if err != nil {
		return nil, err
	}

	return t, nil
}
