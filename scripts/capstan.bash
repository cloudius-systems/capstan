#!bash
#
# bash completion file for core capstan commands
#

__capstan_instances()
{
	local instances="$(capstan instances | awk '{print $1}'  | sed 1d)"
	COMPREPLY=( $( compgen -W "$instances" -- "$cur" ) )
}

__capstan_images()
{
	local images="$(capstan images | awk '{print $1}'  | sed 1d)"
	COMPREPLY=( $( compgen -W "$images" -- "$cur" ) )
}

_capstan_delete()
{
	__capstan_instances
}

_capstan_stop()
{
	__capstan_instances
}

_capstan_rmi()
{
	__capstan_images
}

_capstan_capstan()
{
	case "$cur" in
		-*)
			COMPREPLY=( $( compgen -W "-h -v --help --version" -- "$cur" ) )
			;;
		*)
			COMPREPLY=( $( compgen -W "$commands help" -- "$cur" ) )
			;;
	esac
}

_capstan()
{
	local commands="
			info
			import
			pull
			rmi
			run
			build
			images
			search
			instances
			stop
			delete
		"

	COMPREPLY=()
	local cur words cword
	_get_comp_words_by_ref -n : cur words cword

	local command='capstan'
	local counter=1
	while [ $counter -lt $cword ]; do
		case "${words[$counter]}" in
			-*)
				(( counter++ ))
				;;
			*)
				command="${words[$counter]}"
				break
				;;
		esac
	done

	local completions_func=_capstan_${command}
	declare -F $completions_func >/dev/null && $completions_func

	return 0
}

complete -F _capstan capstan
