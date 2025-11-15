module github.com/gravelight-studio/box-cli

go 1.23

require (
	github.com/gravelight-studio/box/go v0.1.0
	github.com/manifoldco/promptui v0.9.0
	go.uber.org/zap v1.26.0
)

require (
	github.com/chzyer/readline v0.0.0-20180603132655-2972be24d48e // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/sys v0.0.0-20181122145206-62eef0e2fa9b // indirect
)

replace github.com/gravelight-studio/box/go => ../go
