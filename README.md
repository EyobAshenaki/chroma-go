# chroma-go

use the below command start the server for a client/server architecture

`docker run -t --rm -p 8000:8000 --name chromadb -v ./chroma:/chroma/chroma -e IS_PERSISTENT=TRUE -e ANONYMIZED_TELEMETRY=FALSE chromadb/chroma:0.5.0`
