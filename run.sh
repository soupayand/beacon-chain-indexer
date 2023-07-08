clear
echo "|=================================================================================================================================================|"
echo "|                                                                                                                                                 |"
echo "| This project aims at indexing the last 5 finalized epoch data in the database and additionally present some additional metric about the network |"
echo "|                                                                                                                                                 |"
echo "|=================================================================================================================================================|"
sleep 1
echo "Clearing logs........................." 
>error.log
>info.log
sleep 1
echo "Logs cleared"
sleep 1
echo "Building project.........................................."
go build
sleep 1
echo "Build successfull"
echo "Running project..........................................."
./go-beacon-chain-indexer 
