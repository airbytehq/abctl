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

func TestLastErrorLog(t *testing.T) {
	l, err := getLastLogError(strings.NewReader(testLogs))
	if err != nil {
		t.Errorf("unexpected error %s", err)
	}
	expect := "Caused by: io.airbyte.db.check.DatabaseCheckException: Unable to connect to the database."
	if l != expect {
		t.Errorf("expected %q but got %q", expect, l)
	}
}


