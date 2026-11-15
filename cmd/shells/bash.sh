twig() {
    # Any arguments to twig go directly to the binary, and don't presume a directory change (for now)
	if [[ $# -ne 0 ]] ; then
        command twig "$@"
        return
    fi

    local target_dir
    target_dir=$(command twig "$@")
    local exit_code=$?

    if [ $exit_code -eq 0 ] && [ -n "$target_dir" ] && [ -d "$target_dir" ]; then
        builtin cd "$target_dir" || return
    fi
}
