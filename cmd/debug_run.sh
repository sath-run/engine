debugGrpc="localhost"
result=`netstat -aonp|grep :50051| grep LISTEN`
if [ "$result" = "" ]; then
    nameSrv=`cat /etc/resolv.conf | grep nameserver`
    debugGrpc=${nameSrv#*' '}
    echo "debugGrpc is: $debugGrpc"
fi

SATH_MODE=debug go run main.go -debugGrpc=$debugGrpc
