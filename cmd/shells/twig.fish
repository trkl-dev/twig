function twig
    # Default usage: Calling twig without args
    if test (count $argv) -eq 0
        set -l target_dir (command twig)
        set -l exit_code $status

        if test $exit_code -eq 0 -a -n "$target_dir" -a -d "$target_dir"
            builtin cd "$target_dir"; or return
        end

        return
    end

    # Creating a new branch
    if test (count $argv) -eq 1 -a "$argv[1]" = "new"
        set -l target_dir (command twig new)
        set -l exit_code $status

        if test $exit_code -eq 0 -a -n "$target_dir" -a -d "$target_dir"
            builtin cd "$target_dir"; or return
        end

        return
    end

    # Any remaining arguments to twig go directly to the binary
    command twig $argv
end
