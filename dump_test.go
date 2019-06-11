package mysqldump

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestGetTablesOk(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err, "an error was not expected when opening a stub database connection")
	defer db.Close()

	rows := sqlmock.NewRows([]string{"Tables_in_Testdb"}).
		AddRow("Test_Table_1").
		AddRow("Test_Table_2")

	mock.ExpectQuery("^SHOW TABLES$").WillReturnRows(rows)

	data := Data{
		Connection: db,
	}

	result, err := data.getTables()
	assert.NoError(t, err)

	// we make sure that all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet(), "there were unfulfilled expectations")

	assert.EqualValues(t, []string{"Test_Table_1", "Test_Table_2"}, result)
}

func TestIgnoreTablesOk(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err, "an error was not expected when opening a stub database connection")
	defer db.Close()

	rows := sqlmock.NewRows([]string{"Tables_in_Testdb"}).
		AddRow("Test_Table_1").
		AddRow("Test_Table_2")

	mock.ExpectQuery("^SHOW TABLES$").WillReturnRows(rows)

	data := Data{
		Connection:   db,
		IgnoreTables: []string{"Test_Table_1"},
	}

	result, err := data.getTables()
	assert.NoError(t, err)

	// we make sure that all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet(), "there were unfulfilled expectations")

	assert.EqualValues(t, []string{"Test_Table_2"}, result)
}

func TestGetTablesNil(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err, "an error was not expected when opening a stub database connection")

	defer db.Close()

	rows := sqlmock.NewRows([]string{"Tables_in_Testdb"}).
		AddRow("Test_Table_1").
		AddRow(nil).
		AddRow("Test_Table_3")

	mock.ExpectQuery("^SHOW TABLES$").WillReturnRows(rows)

	data := Data{
		Connection: db,
	}

	result, err := data.getTables()
	assert.NoError(t, err)

	// we make sure that all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet(), "there were unfulfilled expectations")

	assert.EqualValues(t, []string{"Test_Table_1", "Test_Table_3"}, result)
}

func TestGetServerVersionOk(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err, "an error was not expected when opening a stub database connection")

	defer db.Close()

	rows := sqlmock.NewRows([]string{"Version()"}).
		AddRow("test_version")

	mock.ExpectQuery("^SELECT version()").WillReturnRows(rows)

	meta := metaData{}

	assert.NoError(t, meta.updateServerVersion(db), "error was not expected while updating stats")

	// we make sure that all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet(), "there were unfulfilled expectations")

	assert.Equal(t, "test_version", meta.ServerVersion)
}

func TestCreateTableSQLOk(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err, "an error was not expected when opening a stub database connection")
	defer db.Close()

	rows := sqlmock.NewRows([]string{"Table", "Create Table"}).
		AddRow("Test_Table", "CREATE TABLE 'Test_Table' (`id` int(11) NOT NULL AUTO_INCREMENT,`s` char(60) DEFAULT NULL, PRIMARY KEY (`id`))ENGINE=InnoDB DEFAULT CHARSET=latin1")

	mock.ExpectQuery("^SHOW CREATE TABLE `Test_Table`$").WillReturnRows(rows)

	data := Data{
		Connection: db,
	}

	table, err := data.createTable("Test_Table")
	assert.NoError(t, err)

	result, err := table.CreateSQL()
	assert.NoError(t, err)

	// we make sure that all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet(), "there were unfulfilled expectations")

	expectedResult := "CREATE TABLE 'Test_Table' (`id` int(11) NOT NULL AUTO_INCREMENT,`s` char(60) DEFAULT NULL, PRIMARY KEY (`id`))ENGINE=InnoDB DEFAULT CHARSET=latin1"

	if !reflect.DeepEqual(result, expectedResult) {
		t.Fatalf("expected %#v, got %#v", expectedResult, result)
	}
}

func TestCreateTableRowValues(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err, "an error was not expected when opening a stub database connection")
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "email", "name"}).
		AddRow(1, "test@test.de", "Test Name 1").
		AddRow(2, "test2@test.de", "Test Name 2")

	mock.ExpectQuery("^SELECT (.+) FROM `test`$").WillReturnRows(rows)

	data := Data{
		Connection: db,
	}

	table, err := data.createTable("test")
	assert.NoError(t, err)

	assert.True(t, table.Next())

	result := table.RowValues()
	assert.NoError(t, table.Err)

	// we make sure that all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet(), "there were unfulfilled expectations")

	assert.EqualValues(t, "('1','test@test.de','Test Name 1')", result)
}

func TestCreateTableValuesSteam(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err, "an error was not expected when opening a stub database connection")
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "email", "name"}).
		AddRow(1, "test@test.de", "Test Name 1").
		AddRow(2, "test2@test.de", "Test Name 2")

	mock.ExpectQuery("^SELECT (.+) FROM `test`$").WillReturnRows(rows)

	data := Data{
		Connection:       db,
		MaxAllowedPacket: 4096,
	}

	table, err := data.createTable("test")
	assert.NoError(t, err)

	s := table.Stream()
	assert.EqualValues(t, "INSERT INTO `test` VALUES ('1','test@test.de','Test Name 1'),('2','test2@test.de','Test Name 2');", <-s)

	// we make sure that all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet(), "there were unfulfilled expectations")
}

func TestCreateTableValuesSteamSmallPackets(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err, "an error was not expected when opening a stub database connection")
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "email", "name"}).
		AddRow(1, "test@test.de", "Test Name 1").
		AddRow(2, "test2@test.de", "Test Name 2")

	mock.ExpectQuery("^SELECT (.+) FROM `test`$").WillReturnRows(rows)

	data := Data{
		Connection:       db,
		MaxAllowedPacket: 64,
	}

	table, err := data.createTable("test")
	assert.NoError(t, err)

	s := table.Stream()
	assert.EqualValues(t, "INSERT INTO `test` VALUES ('1','test@test.de','Test Name 1');", <-s)
	assert.EqualValues(t, "INSERT INTO `test` VALUES ('2','test2@test.de','Test Name 2');", <-s)

	// we make sure that all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet(), "there were unfulfilled expectations")
}

func TestCreateTableAllValuesWithNil(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err, "an error was not expected when opening a stub database connection")
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "email", "name"}).
		AddRow(1, nil, "Test Name 1").
		AddRow(2, "test2@test.de", "Test Name 2").
		AddRow(3, "", "Test Name 3")

	mock.ExpectQuery("^SELECT (.+) FROM `test`$").WillReturnRows(rows)

	data := Data{
		Connection: db,
	}

	table, err := data.createTable("test")
	assert.NoError(t, err)

	results := make([]string, 0)
	for table.Next() {
		row := table.RowValues()
		assert.NoError(t, table.Err)
		results = append(results, row)
	}

	// we make sure that all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet(), "there were unfulfilled expectations")

	expectedResults := []string{"('1',NULL,'Test Name 1')", "('2','test2@test.de','Test Name 2')", "('3','','Test Name 3')"}

	assert.EqualValues(t, expectedResults, results)
}

func TestCreateTableOk(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err, "an error was not expected when opening a stub database connection")

	defer db.Close()

	createTableRows := sqlmock.NewRows([]string{"Table", "Create Table"}).
		AddRow("Test_Table", "CREATE TABLE 'Test_Table' (`id` int(11) NOT NULL AUTO_INCREMENT,`s` char(60) DEFAULT NULL, PRIMARY KEY (`id`))ENGINE=InnoDB DEFAULT CHARSET=latin1")

	createTableValueRows := sqlmock.NewRows([]string{"id", "email", "name"}).
		AddRow(1, nil, "Test Name 1").
		AddRow(2, "test2@test.de", "Test Name 2")

	mock.ExpectQuery("^SHOW CREATE TABLE `Test_Table`$").WillReturnRows(createTableRows)
	mock.ExpectQuery("^SELECT (.+) FROM `Test_Table`$").WillReturnRows(createTableValueRows)

	var buf bytes.Buffer
	data := Data{
		Connection:       db,
		Out:              &buf,
		MaxAllowedPacket: 4096,
	}

	assert.NoError(t, data.getTemplates())

	table, err := data.createTable("Test_Table")
	assert.NoError(t, err)

	data.writeTable(table)

	// we make sure that all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet(), "there were unfulfilled expectations")

	expectedResult := `
--
-- Table structure for table ~Test_Table~
--

DROP TABLE IF EXISTS ~Test_Table~;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
 SET character_set_client = utf8mb4 ;
CREATE TABLE 'Test_Table' (~id~ int(11) NOT NULL AUTO_INCREMENT,~s~ char(60) DEFAULT NULL, PRIMARY KEY (~id~))ENGINE=InnoDB DEFAULT CHARSET=latin1;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table ~Test_Table~
--

LOCK TABLES ~Test_Table~ WRITE;
/*!40000 ALTER TABLE ~Test_Table~ DISABLE KEYS */;
INSERT INTO ~Test_Table~ VALUES ('1',NULL,'Test Name 1'),('2','test2@test.de','Test Name 2');
/*!40000 ALTER TABLE ~Test_Table~ ENABLE KEYS */;
UNLOCK TABLES;
`
	result := strings.Replace(buf.String(), "`", "~", -1)
	assert.Equal(t, expectedResult, result)
}

func TestCreateTableOkSmallPackets(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err, "an error was not expected when opening a stub database connection")

	defer db.Close()

	createTableRows := sqlmock.NewRows([]string{"Table", "Create Table"}).
		AddRow("Test_Table", "CREATE TABLE 'Test_Table' (`id` int(11) NOT NULL AUTO_INCREMENT,`s` char(60) DEFAULT NULL, PRIMARY KEY (`id`))ENGINE=InnoDB DEFAULT CHARSET=latin1")

	createTableValueRows := sqlmock.NewRows([]string{"id", "email", "name"}).
		AddRow(1, nil, "Test Name 1").
		AddRow(2, "test2@test.de", "Test Name 2")

	mock.ExpectQuery("^SHOW CREATE TABLE `Test_Table`$").WillReturnRows(createTableRows)
	mock.ExpectQuery("^SELECT (.+) FROM `Test_Table`$").WillReturnRows(createTableValueRows)

	var buf bytes.Buffer
	data := Data{
		Connection:       db,
		Out:              &buf,
		MaxAllowedPacket: 64,
	}

	assert.NoError(t, data.getTemplates())

	table, err := data.createTable("Test_Table")
	assert.NoError(t, err)

	data.writeTable(table)

	// we make sure that all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet(), "there were unfulfilled expectations")

	expectedResult := `
--
-- Table structure for table ~Test_Table~
--

DROP TABLE IF EXISTS ~Test_Table~;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
 SET character_set_client = utf8mb4 ;
CREATE TABLE 'Test_Table' (~id~ int(11) NOT NULL AUTO_INCREMENT,~s~ char(60) DEFAULT NULL, PRIMARY KEY (~id~))ENGINE=InnoDB DEFAULT CHARSET=latin1;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table ~Test_Table~
--

LOCK TABLES ~Test_Table~ WRITE;
/*!40000 ALTER TABLE ~Test_Table~ DISABLE KEYS */;
INSERT INTO ~Test_Table~ VALUES ('1',NULL,'Test Name 1');
INSERT INTO ~Test_Table~ VALUES ('2','test2@test.de','Test Name 2');
/*!40000 ALTER TABLE ~Test_Table~ ENABLE KEYS */;
UNLOCK TABLES;
`
	result := strings.Replace(buf.String(), "`", "~", -1)
	assert.Equal(t, expectedResult, result)
}
