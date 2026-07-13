_snailrace() {
    local cur prev options command_seen expect_value after_separator word i
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    options="-c -command -label -prepare -n -runs -warmups -interval -index -baseline -f -format -o -output -verbose -show-output -d -duration -width -height -v -version -h -help"

    command_seen=0
    expect_value=0
    after_separator=0
    for ((i = 1; i < COMP_CWORD; i++)); do
        word="${COMP_WORDS[i]}"
        if ((expect_value)); then
            expect_value=0
            continue
        fi
        if ((after_separator)); then
            command_seen=1
            continue
        fi
        case "$word" in
            --) after_separator=1 ;;
            tui) ;;
            -c|-command|-label|-prepare|-n|-runs|-warmups|-interval|-index|-baseline|-f|-format|-o|-output|-d|-duration|-width|-height)
                expect_value=1
                ;;
            -*) ;;
            *) command_seen=1 ;;
        esac
    done

    if ((command_seen)); then
        COMPREPLY=($(compgen -f -- "$cur"))
        return
    fi

    case "$prev" in
        -f|-format)
            COMPREPLY=($(compgen -W "html svg markdown json text" -- "$cur"))
            return
            ;;
        -index)
            COMPREPLY=($(compgen -W "time cpu ram disk" -- "$cur"))
            return
            ;;
        -o|-output)
            COMPREPLY=($(compgen -d -- "$cur"))
            return
            ;;
        -c|-command|-label|-prepare|-n|-runs|-warmups|-interval|-baseline|-d|-duration|-width|-height)
            COMPREPLY=()
            return
            ;;
    esac

    if [[ "$cur" == -* ]]; then
        COMPREPLY=($(compgen -W "$options" -- "$cur"))
    elif [[ $COMP_CWORD -eq 1 ]]; then
        COMPREPLY=($(compgen -W "tui" -c -- "$cur"))
    else
        COMPREPLY=($(compgen -c -- "$cur"))
    fi
}

complete -F _snailrace snailrace
