package cli

import (
	"errors"
	"fmt"
	"os"
)

// runCompletion writes a shell-completion script for `bash` or `zsh`.
// Usage:
//
//	mindforge completion bash >> ~/.bashrc
//	mindforge completion zsh  >> ~/.zshrc
//
// The scripts are intentionally tiny: completion lists top-level
// commands and a handful of common subcommand keywords.  No external
// dependency, no codegen — just a static list updated alongside the
// dispatcher in `Run`.
func runCompletion(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: mindforge completion <bash|zsh|fish>")
	}
	switch args[0] {
	case "bash":
		fmt.Fprint(os.Stdout, bashCompletion)
	case "zsh":
		fmt.Fprint(os.Stdout, zshCompletion)
	case "fish":
		fmt.Fprint(os.Stdout, fishCompletion)
	default:
		return fmt.Errorf("unsupported shell %q (try bash, zsh, fish)", args[0])
	}
	return nil
}

// completionTopLevel keeps the canonical list of commands a user can
// type as the first argument.  Update it whenever you add a new
// dispatcher entry in `Run`.  The shell scripts below interpolate it.
var completionTopLevel = []string{
	"add", "agenda", "backup", "calendar", "completion", "config", "doctor",
	"edit-data", "export", "habit", "help", "import", "journal", "motd",
	"note", "pomodoro", "quote", "restore", "review", "search", "stats",
	"summary", "tags", "task", "today", "undo", "vault", "version", "welcome",
	"where",
}

const bashCompletion = `# mindforge bash completion — append to ~/.bashrc or source on demand
_mindforge_complete() {
    local cur prev cmds
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    cmds="add agenda backup calendar completion config doctor edit-data export habit help import journal motd note pomodoro quote restore review search stats summary tags task today undo vault version welcome where"
    if [ ${COMP_CWORD} -eq 1 ]; then
        COMPREPLY=( $(compgen -W "${cmds}" -- ${cur}) )
        return 0
    fi
    case "${COMP_WORDS[1]}" in
        note|notes|n) COMPREPLY=( $(compgen -W "add list show edit pin unpin delete" -- ${cur}) ) ;;
        task|tasks|t) COMPREPLY=( $(compgen -W "add list mark edit delete archive tidy repeat next pri due" -- ${cur}) ) ;;
        journal|j) COMPREPLY=( $(compgen -W "write mood show log trend" -- ${cur}) ) ;;
        habit|habits|h) COMPREPLY=( $(compgen -W "add list check uncheck delete report" -- ${cur}) ) ;;
        pomodoro|pomo|p) COMPREPLY=( $(compgen -W "--label --rounds --minutes --break" -- ${cur}) ) ;;
        tags) COMPREPLY=( $(compgen -W "rename delete" -- ${cur}) ) ;;
        completion) COMPREPLY=( $(compgen -W "bash zsh fish" -- ${cur}) ) ;;
        export) COMPREPLY=( $(compgen -W "--format md csv json" -- ${cur}) ) ;;
        config) COMPREPLY=( $(compgen -W "--get --set" -- ${cur}) ) ;;
        vault) COMPREPLY=( $(compgen -W "lock unlock show" -- ${cur}) ) ;;
        *) COMPREPLY=( ) ;;
    esac
}
complete -F _mindforge_complete mindforge
`

const zshCompletion = `# mindforge zsh completion — append to ~/.zshrc or source on demand
_mindforge() {
    local -a cmds
    cmds=(add agenda backup calendar completion config doctor edit-data export habit help import journal motd note pomodoro quote restore review search stats summary tags task today undo vault version welcome where)
    if (( CURRENT == 2 )); then
        _describe 'mindforge command' cmds
        return
    fi
    case ${words[2]} in
        note|notes|n) _values 'note subcommand' add list show edit pin unpin delete ;;
        task|tasks|t) _values 'task subcommand' add list mark edit delete archive tidy repeat next pri due ;;
        journal|j) _values 'journal subcommand' write mood show log trend ;;
        habit|habits|h) _values 'habit subcommand' add list check uncheck delete report ;;
        tags) _values 'tag subcommand' rename delete ;;
        completion) _values 'shell' bash zsh fish ;;
        vault) _values 'vault subcommand' lock unlock show ;;
    esac
}
compdef _mindforge mindforge
`

const fishCompletion = `# mindforge fish completion — save under ~/.config/fish/completions/mindforge.fish
set -l mf_cmds add agenda backup calendar completion config doctor edit-data export habit help import journal motd note pomodoro quote restore review search stats summary tags task today undo vault version welcome where
complete -f -c mindforge -n "not __fish_seen_subcommand_from $mf_cmds" -a "$mf_cmds"
complete -f -c mindforge -n "__fish_seen_subcommand_from note notes n" -a "add list show edit pin unpin delete"
complete -f -c mindforge -n "__fish_seen_subcommand_from task tasks t" -a "add list mark edit delete archive tidy repeat next pri due"
complete -f -c mindforge -n "__fish_seen_subcommand_from journal j" -a "write mood show log trend"
complete -f -c mindforge -n "__fish_seen_subcommand_from habit habits h" -a "add list check uncheck delete report"
complete -f -c mindforge -n "__fish_seen_subcommand_from tags" -a "rename delete"
complete -f -c mindforge -n "__fish_seen_subcommand_from completion" -a "bash zsh fish"
complete -f -c mindforge -n "__fish_seen_subcommand_from vault" -a "lock unlock show"
`
