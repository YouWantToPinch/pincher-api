#!/bin/bash

# THIS TOOL IS USED FOR building tests for the pincher-api.
# With the added user-friendliness of Charm's `gum` glamorizing tool,
#  you can easily build scripts for testing the pincher webserver api.
# Simply build each line of your test script using the prompts exposed
#  to you by the tool, which knows exactly what arguments are required
#  by the commands you want to run, so that you needn't keep track of that.
# Of course, there are no checks by the tool as to what
#  theoretically should or should not be possible; otherwise the purpose
#  of the tool would be defeated.
# ! IMPORTANT: Output scripts expect for their possible CRUD functions to be
#  defined within a file in the same directory entitled 'functions'.

declare -r COMMANDS=("Add" "Page" "Finish" "Cancel" "Undo")
declare -r ENTITIES=("*DEBUG" "+HEADING" "categories" "groups" "users")
# declare -r ENTITIES=("users" "groups" "categories" "accounts" "transactions")
declare -r USER_ACTIONS=("create get delete login reset") # TODO: add logout...
declare -r GROUP_ACTIONS=("create get delete")
declare -r CATEGORY_ACTIONS=("create get delete assign")
# declare -r ACCOUNT_ACTIONS=("create" "get" "delete")
# declare -r TRANSACTION_ACTIONS=("create" "get" "delete")

declare -A action_map
action_map["users"]="USER_ACTIONS"
action_map["groups"]="GROUP_ACTIONS"
action_map["categories"]="CATEGORY_ACTIONS"
#action_map["accounts"]="ACCOUNT_ACTIONS"
#action_map["transactions"]="TRANSACTION_ACTIONS"

declare -A TEST_ENTITY_COUNTER
declare -A TEST_ENTITY_LISTS

GLO_SAVED_ENTITY_VAR=""

# ----- FUNCTIONS -----

parse_en_ls() {
    type="$1"
    ARRAY_NAME="${TEST_ENTITY_LISTS[$type]}"
    ACTION=$(gum choose --ordered ${!ARRAY_NAME})
}

users_login() {
    local username=$(gum input --placeholder "Enter a username...")
    local password=$(gum input --placeholder "Enter a password...")
    add_bash_line "SAVE" "JWT_USER" "login" "$username" "$password"
}

users_create() {
    local username=$(gum input --placeholder "New username...")
    local password=$(gum input --placeholder "New password...")
    add_bash_line "SAVE" "USER" "create_user" "$username" "$password"
}

users_reset() {
    add_bash_line "EXEC" "USER" "reset"
}

users_delete() {
    local token=$(gum choose ${TEST_ENTITY_LISTS["JWT_USER"]})
    local username=$(gum input --placeholder "Enter a username...")
    local password=$(gum input --placeholder "Enter a password...")
    add_bash_line "EXEC" "USER" "delete_user" ""\$$token"" "$username" "$password"
}

groups_create() {
    local token=$(gum choose ${TEST_ENTITY_LISTS["JWT_USER"]})
    local user_id=$(gum choose ${TEST_ENTITY_LISTS["USER"]})
    local name=$(gum input --placeholder "Name for new group...")
    local notes=$(gum write --placeholder "Write note(s) about group...")
    add_bash_line "SAVE" "GROUP" "create_group" "\$$token" "\$$user_id" "$name" "$notes"
}

groups_get() {
    local token=$(gum choose ${TEST_ENTITY_LISTS["JWT_USER"]})
    local user_id=$(gum choose ${TEST_ENTITY_LISTS["USER"]})
    #local query=$(gum input --placeholder "Optional query param (?key=value)...")
    add_bash_line "GET" "GROUP" "get_user_groups" "\$$token" "\$$user_id" "$query"
}

groups_delete() {
    local token=$(gum choose ${TEST_ENTITY_LISTS["JWT_USER"]})
    local user_id=$(gum choose ${TEST_ENTITY_LISTS["USER"]})
    local group_id=$(gum choose ${TEST_ENTITY_LISTS["GROUP"]})
    add_bash_line "EXEC" "GROUP" "delete_user_group" "\$$token" "\$$user_id" "\$$group_id"
}

categories_create() {
    local token=$(gum choose ${TEST_ENTITY_LISTS["JWT_USER"]})
    local user_id=$(gum choose ${TEST_ENTITY_LISTS["USER"]})
    local group_id=$(gum choose ${TEST_ENTITY_LISTS["GROUP"]})
    local name=$(gum input --placeholder "Name for new category...")
    local notes=$(gum write --placeholder "Note(s) for new category...")
    add_bash_line "SAVE" "CATEGORY" "create_category" "\$$token" "\$$user_id" "\$$group_id" "$name" "$notes"
}

categories_get() {
    local token=$(gum choose ${TEST_ENTITY_LISTS["JWT_USER"]})
    local user_id=$(gum choose ${TEST_ENTITY_LISTS["USER"]})
    local query=$(gum input --placeholder "Optional query param (?key=value)...")
    add_bash_line "GET" "CATEGORY" "get_user_categories" "\$$token" "\$$user_id" "$query"
}

categories_delete() {
    local token=$(gum choose ${TEST_ENTITY_LISTS["JWT_USER"]})
    local user_id=$(gum choose ${TEST_ENTITY_LISTS["USER"]})
    local category_id=$(gum choose ${TEST_ENTITY_LISTS["CATEGORY"]})
    add_bash_line "SAVE" "CATEGORY" "delete_user_category" "\$$token" "\$$user_id" "\$$category_id"
}

categories_assign() {
    local token=$(gum choose ${TEST_ENTITY_LISTS["JWT_USER"]})
    local user_id=$(gum choose ${TEST_ENTITY_LISTS["USER"]})
    local category_id=$(gum choose ${TEST_ENTITY_LISTS["CATEGORY"]})
    local group_id=$(gum choose ${TEST_ENTITY_LISTS["GROUP"]})
    add_bash_line "EXEC" "CATEGORY" "assign_cat_to_grp" "\$$token" "\$$user_id" "\$$category_id" "\$$group_id"
}

clean_unused_entities() {
    local file="temp_test_script"
    local new_list

    for type in "${!TEST_ENTITY_LISTS[@]}"; do
        new_list=""
        # Read entities into array
        IFS=' ' read -ra entities <<< "${TEST_ENTITY_LISTS[$type]}"
        for entity in "${entities[@]}"; do
            # Check if this entity name appears in the file
            if grep -q "\b$entity\b" "$file"; then
                new_list+="$entity "
            fi
        done
        # Trim trailing space and update
        TEST_ENTITY_LISTS[$type]="${new_list% }"
    done
}

save_entity() {
    local type="$1"
    shift
    local args=("$@")

    local quoted_args=()
    for arg in "${args[@]}"; do
        quoted_args+=("\"$arg\"")
    done

    (( TEST_ENTITY_COUNTER[$type]++ ))
    local index="${TEST_ENTITY_COUNTER[$type]}"
    local varname="${type^^}${index}"

    TEST_ENTITY_LISTS["$type"]+="${varname} "

    GLO_SAVED_ENTITY_VAR="$varname"
}

add_bash_line() {
    local should_save="$1"
    local type="$2"
    shift 2
    local cmd="$1"
    shift
    local args=("$@")

    local quoted_args=()
    for arg in "${args[@]}"; do
        quoted_args+=("\"$arg\"")
    done

    if [[ $should_save == "SAVE" ]]; then
        save_entity "$type"
        echo "$GLO_SAVED_ENTITY_VAR=\$($cmd ${quoted_args[*]})"
    else
        echo "$cmd ${quoted_args[*]}"
    fi
}

list_entities() {
    local type="$1"
    echo "${ENTITY_LISTS[$type]}"
}

# ------ SCRIPT -------

if ! command -v gum >/dev/null 2>&1
then
    echo "ERROR: Gum not installed."
    echo "Please install gum before running the script."
    exit 1
fi

gum style \
	--foreground 212 --border-foreground 212 --border double \
	--align center --width 50 --margin "1 2" --padding "2 4" \
	'Add script commands with ADD.' \
    'Read through your progress with PAGE.' \
    'Save your script with FINISH.' \
    'Cancel writing your script with CANCEL.' \
    'Remove the last line with UNDO.' \

# Loop to gather user input
while true; do
    CHOICE=$(gum choose --ordered ${COMMANDS[@]})
    if [[ "$CHOICE" == "Undo" ]]; then
        if [[ $(grep -c "^" temp_test_script) -gt 1 ]]; then
            sed -i '$d' "temp_test_script"
            clean_unused_entities
        fi
    elif [[ "$CHOICE" == "Cancel" ]]; then
        gum confirm && rm -f temp_test_script && exit 0
    elif [[ "$CHOICE" == "Page" ]]; then
        gum pager < temp_test_script
        continue
    elif [[ "$CHOICE" == "Finish" ]]; then
        gum confirm && break
    elif [[ "$CHOICE" == "Add" ]]; then
        ENTITY="NONE"
        ACTION="NONE"
        ENTITY=$(gum choose --ordered ${ENTITIES[@]})
        if [[ "$ENTITY" != "NONE" ]]; then
            if [[ "$ENTITY" == "+HEADING" ]]; then
                HEADING=$(gum input --placeholder "Test log output for endpoint call...")
                echo "heading \"$HEADING\"" >> "temp_test_script"
                continue
            elif [[ "$ENTITY" == "*DEBUG" ]]; then
                gum log --structured --level debug "T_EN_LS[USR]: ${TEST_ENTITY_LISTS["USER"]}"
                gum log --structured --level debug "T_EN_LS[JWT]: ${TEST_ENTITY_LISTS["JWT_USER"]}"
            else
                ARRAY_NAME="${action_map[$ENTITY]}"
                ACTION=$(gum choose --ordered ${!ARRAY_NAME})
                if [[ "$ACTION" != "NONE" ]]; then
                    CALL_CMD="${ENTITY}_${ACTION}"
                    $CALL_CMD >> "temp_test_script"
                fi
            fi
        fi
    fi
done

test_script=$(gum input --placeholder "Name of your test script...")
test_description=$(gum write --placeholder "Description of your test script...")

gum spin --spinner dot --title "Finalizing $test_script..." -- sleep 5

echo "#!/bin/bash" > "$test_script"
echo "#$test_description"  >> "$test_script"
echo "source ./functions" >> "$test_script"
echo "" >> "$test_script"

cat temp_test_script >> "$test_script"
rm temp_test_script

chmod +x "$test_script"

mv "$test_script" ../tests/

echo "Test script '$test_script' built successfully!"
