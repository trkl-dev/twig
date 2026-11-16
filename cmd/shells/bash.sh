twig() {
    # Default usage: Calling twig without args
	if [[ $# -eq 0 ]] ; then
        local target_dir
        target_dir=$(command twig)
        local exit_code=$?

        if [ $exit_code -eq 0 ] && [ -n "$target_dir" ] && [ -d "$target_dir" ]; then
            builtin cd "$target_dir" || return
        fi

        return
    fi

    # Creating a new branch
    if [[ "$1" == "new"  &&  $# -eq 1 ]]; then
        local target_dir
        target_dir=$(command twig new)
        local exit_code=$?

        if [ $exit_code -eq 0 ] && [ -n "$target_dir" ] && [ -d "$target_dir" ]; then
            builtin cd "$target_dir" || return
        fi

        return
    fi

    # Any remaining arguments to twig go directly to the binary, and don't presume a directory change (for now)
    command twig "$@"
}
