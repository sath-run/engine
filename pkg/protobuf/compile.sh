dir=$(pwd)
SCRIPT_DIR="$( cd -- "$( dirname -- "${BASH_SOURCE[0]:-$0}"; )" &> /dev/null && pwd 2> /dev/null; )";

cd $SCRIPT_DIR
rm -f *.pb.go

protoc --proto_path=. \
   --go_out=. --go_opt=paths=source_relative  \
   --go-grpc_out=. --go-grpc_opt=paths=source_relative \
   *.proto
cd $dir
