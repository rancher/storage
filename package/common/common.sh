set -e -o pipefail

err() {
    echo -ne $@ 1>&2
}

usage() {
    err "Usage: "
    err "\t$0 create <json params>"
    err "\t$0 delete <json params>"
    err "\t$0 attach <json params>"
    err "\t$0 detach <device>"
    err "\t$0 mount <mount dir> <device> <json params>"
    err "\t$0 unmount <mount dir> <json params>"
    err "\t$0 init"
    exit 1
}

main()
{

    case $1 in
        init)
            "$@"
            ;;
        create|delete|attach)
            parse "$2"
            "$@"
            ;;
        detach)
            DEVICE="$2"
            "$@"
            ;;
        mount)
            MNT_DEST="$2"
            DEVICE="$3"
            parse "$4"
            shift 1
            mountdest "$@"
            ;;
        unmount)
            MNT_DEST="$2"
            parse "$3"
            "$@"
            ;;
        *)
            usage
            ;;
    esac
}

declare -A OPTS
parse()
{
    mapfile -t < <(echo "$1" | jq -r 'to_entries | map([.key, .value]) | .[]' | jq '.[]' | sed 's!^"\(.*\)"$!\1!g')
    for ((i=0;i < ${#MAPFILE[@]} ; i+=2)) do
        OPTS[${MAPFILE[$i]}]=${MAPFILE[$((i+1))]}
    done
}

print_options()
{
    for ((i=1; i < $#; i+=2)) do
        j=$((i+1))
        jq -n --arg k ${!i} --arg v ${!j} '{"key": $k, "value": $v}'
    done | jq -c -s '{"status": "Success", "options": from_entries}'
}

print_device()
{
    echo -n "$@" | jq -R -c -s '{"status": "Success", "device": .}'
}

print_not_supported()
{
    echo -n "$@" | jq -R -c -s '{"status": "Not supported", "message": .}'
}

print_success()
{
    echo -n "$@" | jq -R -c -s '{"status": "Success", "message": .}'
}

print_error()
{
    echo -n "$@" | jq -R -c -s '{"status": "Failure", "message": .}'
    return 1
}

ismounted() {
    local mountPoint=$1
    local mountP=`findmnt -n ${mountPoint} 2>/dev/null | cut -d' ' -f1`
    if [ "${mountP}" == "${mountPoint}" ]; then
        echo "1"
    else
        echo "0"
    fi
}

init_nfs_client_service() {
    # using host network context to start rpcbind and rpc.statd inside the container process.
    # this requires mapping host pid namespace to this containers when container starts.
    # here parent is the storage --driver rancher-nfs process, the script process is launched
    # on demand of each create/delete etc calls, so host process pid is 2 hops away
    PARENT_PID=$(ps --no-header --pid $$ -o ppid)
    TARGET_PID=$(ps --no-header --pid ${PARENT_PID} -o ppid)
    nsenter -t $TARGET_PID -n rpcbind >& /dev/null
    nsenter -t $TARGET_PID -n rpc.statd >& /dev/null
}

get_host_process_pid() {
    PARENT_PID=$(ps --no-header --pid $$ -o ppid)
    TARGET_PID=$(ps --no-header --pid ${PARENT_PID} -o ppid)
}

unset_aws_credentials_env() {
    if [ -z "${AWS_ACCESS_KEY_ID}" ] || [ -z "${AWS_SECRET_ACCESS_KEY}" ]; then
        unset AWS_ACCESS_KEY_ID
        unset AWS_SECRET_ACCESS_KEY
    fi
}