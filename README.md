# kudzu

Rebuild server for GROW. Provides equivalent API but changes how we do indexing.

## Production deployment

Environment variables to put into Chamber:

KUDZU_DATABASE_URL
KUDZU_THINGFUL_URL
KUDZU_THINGFUL_KEY

Command line invocation for now:

kudzu server --no-indexer