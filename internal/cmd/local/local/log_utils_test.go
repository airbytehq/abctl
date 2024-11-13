package local

import (
	"strings"
	"testing"
)

var testLogs = strings.TrimSpace(`
2024-09-12 15:56:25 [32mINFO[m i.a.d.c.DatabaseAvailabilityCheck(check):49 - Database is not ready yet. Please wait a moment, it might still be initializing...
2024-09-12 15:56:30 [33mWARN[m i.m.s.r.u.Loggers$Slf4JLogger(warn):299 - [54bd6014, L:/127.0.0.1:52991 - R:localhost/127.0.0.1:8125] An exception has been observed post termination, use DEBUG level to see the full stack: java.net.PortUnreachableException: recvAddress(..) failed: Connection refused
2024-09-12 15:56:31 [1;31mERROR[m i.a.b.Application(main):25 - Unable to bootstrap Airbyte environment.
io.airbyte.db.init.DatabaseInitializationException: Database availability check failed.
	at io.airbyte.db.init.DatabaseInitializer.initialize(DatabaseInitializer.java:54) ~[io.airbyte.airbyte-db-db-lib-0.64.3.jar:?]
	at io.airbyte.bootloader.Bootloader.initializeDatabases(Bootloader.java:229) ~[io.airbyte-airbyte-bootloader-0.64.3.jar:?]
	at io.airbyte.bootloader.Bootloader.load(Bootloader.java:104) ~[io.airbyte-airbyte-bootloader-0.64.3.jar:?]
	at io.airbyte.bootloader.Application.main(Application.java:22) [io.airbyte-airbyte-bootloader-0.64.3.jar:?]
Caused by: io.airbyte.db.check.DatabaseCheckException: Unable to connect to the database.
	at io.airbyte.db.check.DatabaseAvailabilityCheck.check(DatabaseAvailabilityCheck.java:40) ~[io.airbyte.airbyte-db-db-lib-0.64.3.jar:?]
	at io.airbyte.db.init.DatabaseInitializer.initialize(DatabaseInitializer.java:45) ~[io.airbyte.airbyte-db-db-lib-0.64.3.jar:?]
	... 3 more
2024-09-12 15:56:31 [32mINFO[m i.m.r.Micronaut(lambda$start$0):118 - Embedded Application shutting down
2024-09-12T15:56:33.125352208Z Thread-4 INFO Loading mask data from '/seed/specs_secrets_mask.yaml
`)

var testLogs2 = strings.TrimSpace(`
2024-11-12 20:52:07,403 [main]	[1;31mERROR[0;39m	i.a.d.c.DatabaseAvailabilityCheck(lambda$isDatabaseConnected$1):78 - Failed to verify database connection.
org.jooq.exception.DataAccessException: Error getting connection from data source HikariDataSource (HikariPool-1)
	at org.jooq_3.19.7.POSTGRES.debug(Unknown Source)
	at org.jooq.impl.DataSourceConnectionProvider.acquire(DataSourceConnectionProvider.java:90)
Caused by: java.sql.SQLTransientConnectionException: HikariPool-1 - Connection is not available, request timed out after 30110ms (total=0, active=0, idle=0, waiting=0)
	at com.zaxxer.hikari.pool.HikariPool.createTimeoutException(HikariPool.java:686)
	at com.zaxxer.hikari.pool.HikariPool.getConnection(HikariPool.java:179)
	at com.zaxxer.hikari.pool.HikariPool.getConnection(HikariPool.java:144)
	at com.zaxxer.hikari.HikariDataSource.getConnection(HikariDataSource.java:99)
	at org.jooq.impl.DataSourceConnectionProvider.acquire(DataSourceConnectionProvider.java:87)
	... 22 common frames omitted
Caused by: org.postgresql.util.PSQLException: The connection attempt failed.
	at org.postgresql.core.v3.ConnectionFactoryImpl.openConnectionImpl(ConnectionFactoryImpl.java:364)
	at org.postgresql.core.ConnectionFactory.openConnection(ConnectionFactory.java:54)
Caused by: java.net.UnknownHostException: airbyte-db-svc
	at java.base/sun.nio.ch.NioSocketImpl.connect(NioSocketImpl.java:567)
	at org.postgresql.core.v3.ConnectionFactoryImpl.openConnectionImpl(ConnectionFactoryImpl.java:268)
	... 14 common frames omitted
`)

func TestJavaLogScanner(t *testing.T) {
	s := newLogScanner(strings.NewReader(testLogs))

	expectLogLine := func(level, msg string) {
		s.Scan()

		if s.line.level != level {
			t.Errorf("expected level %q but got %q", level, s.line.level)
		}
		if s.line.msg != msg {
			t.Errorf("expected msg %q but got %q", msg, s.line.msg)
		}
		if s.Err() != nil {
			t.Errorf("unexpected error %v", s.Err())
		}
	}

	expectLogLine("INFO", "i.a.d.c.DatabaseAvailabilityCheck(check):49 - Database is not ready yet. Please wait a moment, it might still be initializing...")
	expectLogLine("WARN", "i.m.s.r.u.Loggers$Slf4JLogger(warn):299 - [54bd6014, L:/127.0.0.1:52991 - R:localhost/127.0.0.1:8125] An exception has been observed post termination, use DEBUG level to see the full stack: java.net.PortUnreachableException: recvAddress(..) failed: Connection refused")
	expectLogLine("ERROR", "i.a.b.Application(main):25 - Unable to bootstrap Airbyte environment.")
	expectLogLine("ERROR", "io.airbyte.db.init.DatabaseInitializationException: Database availability check failed.")
	expectLogLine("ERROR", "Caused by: io.airbyte.db.check.DatabaseCheckException: Unable to connect to the database.")
	expectLogLine("INFO", "i.m.r.Micronaut(lambda$start$0):118 - Embedded Application shutting down")
	expectLogLine("INFO", "2024-09-12T15:56:33.125352208Z Thread-4 INFO Loading mask data from '/seed/specs_secrets_mask.yaml")
}
