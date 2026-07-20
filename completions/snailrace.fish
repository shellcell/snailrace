complete -c snailrace -s c -l command -r -d 'Add shell command'
complete -c snailrace -l label -r -d 'Set command label'
complete -c snailrace -l prepare -r -d 'Run one-time setup command'
complete -c snailrace -s n -l runs -r -d 'Set measured runs'
complete -c snailrace -l warmups -r -d 'Set warmup runs'
complete -c snailrace -l interval -r -d 'Set sampling interval'
complete -c snailrace -l index -r -a 'time cpu ram disk' -d 'Set index dimensions'
complete -c snailrace -l baseline -r -d 'Set 1-based baseline'
complete -c snailrace -s f -l format -r -a 'html svg markdown json text' -d 'Select report format'
complete -c snailrace -s o -l output -r -a '(__fish_complete_directories)' -d 'Set report directory'
complete -c snailrace -l no-save -d 'Do not save report files'
complete -c snailrace -l verbose -d 'Print full statistical tables'
complete -c snailrace -l show-output -d 'Forward command output'
complete -c snailrace -s d -l duration -r -d 'Set fixed TUI duration'
complete -c snailrace -l width -r -d 'Set TUI columns'
complete -c snailrace -l height -r -d 'Set TUI rows'
complete -c snailrace -s v -l version -d 'Print version'
complete -c snailrace -s h -l help -d 'Show help'
complete -c snailrace -n '__snailrace_needs_command; and not __fish_seen_subcommand_from tui' -a tui -d 'Run inside a pseudo-terminal'
complete -c snailrace -n '__snailrace_needs_command' -a '(__fish_complete_command)'
function __snailrace_needs_command
    set -l tokens (commandline -opc)
    set -e tokens[1]
    set -l expect_value 0
    set -l after_separator 0
    for token in $tokens
        if test $expect_value -eq 1
            set expect_value 0
            continue
        end
        if test $after_separator -eq 1
            return 1
        end
        switch $token
            case '--'
                set after_separator 1
            case tui
                continue
            case -c -command --command -label --label -prepare --prepare -n -runs --runs -warmups --warmups -interval --interval -index --index -baseline --baseline -f -format --format -o -output --output -d -duration --duration -width --width -height --height
                set expect_value 1
            case '-*'
                continue
            case '*'
                return 1
        end
    end
    test $expect_value -eq 0
end
