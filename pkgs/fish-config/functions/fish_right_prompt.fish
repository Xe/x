function fish_right_prompt
    set -l st $status

    if [ $status != 0 ]
        echo (set_color $theme_color_error) ↵ $st(set_color $theme_color_normal)
    end
end
