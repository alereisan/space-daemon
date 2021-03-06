package sqlite

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/FleekHQ/space-daemon/core/search"

	"gotest.tools/assert"
)

func setupEngine(t *testing.T) (*sqliteFilesSearchEngine, context.Context) {
	dbPath, err := ioutil.TempDir("", "testDb-*")
	assert.NilError(t, err, "failed to create db path")

	engine := NewSearchEngine(WithDBPath(dbPath))
	assert.NilError(t, engine.Start(), "database failed to initialize")

	cleanup := func() {
		_ = engine.Shutdown()
		_ = os.RemoveAll(dbPath)
	}

	t.Cleanup(cleanup)

	return engine, context.Background()
}

func TestSqliteFilesSearchEngine_Insert_And_Query(t *testing.T) {
	engine, ctx := setupEngine(t)
	insertRecord(t, ctx, engine, &search.InsertIndexRecord{
		ItemName:      "new content.pdf",
		ItemExtension: "pdf",
		ItemPath:      "/new",
		ItemType:      "FILE",
		BucketSlug:    "personal",
		DbId:          "",
	})
	insertRecord(t, ctx, engine, &search.InsertIndexRecord{
		ItemName:      "second-content.txt",
		ItemExtension: "txt",
		ItemPath:      "/new",
		ItemType:      "FILE",
		BucketSlug:    "personal",
		DbId:          "",
	})

	queryResult, err := engine.QueryFileData(ctx, "pdf", 20)
	assert.NilError(t, err, "failed to query file data")
	assert.Equal(t, 1, len(queryResult), "not enough results returned from query")

	assert.Equal(t, "new content.pdf", queryResult[0].ItemName, "search query result incorrect")
}

func TestInserting_DuplicateRecords_Fail(t *testing.T) {
	engine, ctx := setupEngine(t)
	insertRecord(t, ctx, engine, &search.InsertIndexRecord{
		ItemName:      "new content.pdf",
		ItemExtension: "pdf",
		ItemPath:      "/new",
		ItemType:      "FILE",
		BucketSlug:    "personal",
		DbId:          "",
	})
	// try inserting duplicate records should fail
	_, err := engine.InsertFileData(ctx, &search.InsertIndexRecord{
		ItemName:      "new content.pdf",
		ItemExtension: "pdf",
		ItemPath:      "/new",
		ItemType:      "FILE",
		BucketSlug:    "personal",
		DbId:          "",
	})
	assert.Error(t, err, "a similar file has already been inserted")
}

func TestSqliteFilesSearchEngine_Delete_And_Query(t *testing.T) {
	engine, ctx := setupEngine(t)
	insertRecord(t, ctx, engine, &search.InsertIndexRecord{
		ItemName:      "new content.pdf",
		ItemExtension: "pdf",
		ItemPath:      "/new",
		ItemType:      "FILE",
		BucketSlug:    "personal",
		DbId:          "",
	})
	insertRecord(t, ctx, engine, &search.InsertIndexRecord{
		ItemName:      "second-content.txt",
		ItemExtension: "txt",
		ItemPath:      "/new",
		ItemType:      "FILE",
		BucketSlug:    "personal",
		DbId:          "",
	})

	err := engine.DeleteFileData(ctx, &search.DeleteIndexRecord{
		ItemName:   "new content.pdf",
		ItemPath:   "/new",
		BucketSlug: "personal",
	})
	assert.NilError(t, err, "deleting file data failed")

	queryResult, err := engine.QueryFileData(ctx, "content", 20)
	assert.NilError(t, err, "failed to query file data")
	assert.Equal(t, 1, len(queryResult), "too much result returned")

	// only second content should exist in search engine
	assert.Equal(t, "second-content.txt", queryResult[0].ItemName, "search query result incorrect")
}

func insertRecord(
	t *testing.T,
	ctx context.Context,
	engine search.FilesSearchEngine,
	record *search.InsertIndexRecord,
) {
	_, err := engine.InsertFileData(ctx, record)
	assert.NilError(t, err, "failed to insert file data")
}
