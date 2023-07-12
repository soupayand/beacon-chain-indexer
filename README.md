# go-beacon-chain-indexer

# **Setup Steps**
1. The first step is to setup the database. For this, I have used TimeScaleDB and the steps to setup a hosted TimeScaleDB service can be found on this url: https://docs.timescale.com/getting-started/latest/services/
2. Replace the database url in .env file with the one obtained after creating this database.
3. Login to the database CLI and run the db.sql file to create the schema.
4. Run run.sh file to start the server.
5. Upon Starting the data from last 5 finalized epoch would be indexed/loaded into the beacon_chain_data table.

# **API endpoints**:
1. GET : /data => This can be used to fetch all the indexed data from the database and filtered on any one of the fields at a time
2. GET : /data?epoch=${EPOCH_NUMBER}&slot={$SLOT_NUMBER}&unix_time=${UNIX_TIME} => This endpoint can be used to filter the indexed data on any one of the fields
3. GET : /participation-rate?epoch=${NO_OF_EPOCHS} => This endpoint can be used to fetch the total participation rate of the validators over the specific no of epochs
4. GET : /participation-rate?epoch=${NO_OF_EPOCHS}&validatorIndex=${INDEX_OF_VALIDATOR} => This can be used to fetch the participation rate for a particular validator over the specific no of epochs

# **Considerations**:
1. Since this was a small project the functionality has been given priority of performance. While performance isn't necessarily poor, it could be optimised nonetheless.
2. To facilitate higher performance, Go routines have been used to fetch data from the quicknode APIs.
3. Only 25 requests/second is currently allowed by Quicknode in the free plan, so rate limits had to be put even where go routines were used to fetch data.
4. Another optimisation mechanism could be to fetch and store the participation data in the database and use queries to generate the desired output.
5. Caching can be used for the participation-rates which when determined for a particular epoch can be stored and quickly retrieved.
6. Event listeners can be used to fetch live data instead of fetching data for a specific no of epochs
7. Time compelling these are some future works that can be undertaken to enhance the solution.
