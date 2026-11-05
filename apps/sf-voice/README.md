# sf-voice JVM examples

Java/Spring Boot and Kotlin/Ktor REST proxy examples for the JVM SDKs. The JVM
SDKs target Java 17.

## setup

```bash
cp .env.example .env
# fill in SF_VOICE_API_KEY
```

Build the local SDK jars first:

```bash
(cd ../../java && gradle jar)
(cd ../../kotlin && gradle jar)
```

## NOTE

If Gradle fails before evaluating the project with native-cache or socket
errors, run these checks outside the restricted sandbox. A fresh Gradle home can
also avoid stale native-service cache issues:

```bash
export GRADLE_USER_HOME=/private/tmp/sfvoice-gradle-home
```

From this package:

```bash
(cd ../../java && gradle jar --no-daemon --quiet)
(cd ../../kotlin && gradle jar --no-daemon --quiet)
gradle :java-example:compileJava --no-daemon --quiet
gradle :kotlin-example:compileKotlin --no-daemon --quiet
```

## smoke check

```bash
gradle :java-example:compileJava
gradle :kotlin-example:compileKotlin
```

## run

```bash
set -a; source .env; set +a
gradle :java-example:bootRun
```

```bash
set -a; source .env; set +a
gradle :kotlin-example:run
```
