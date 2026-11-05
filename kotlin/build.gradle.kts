plugins {
    kotlin("jvm") version "2.0.0"
    kotlin("plugin.serialization") version "2.0.0"
    `java-library`
    `maven-publish`
}

group = "sh.sf-voice"
version = "0.1.1"

repositories {
    mavenCentral()
}

dependencies {
    implementation("io.ktor:ktor-client-core:3.0.0")
    implementation("io.ktor:ktor-client-cio:3.0.0")
    implementation("io.ktor:ktor-client-content-negotiation:3.0.0")
    implementation("io.ktor:ktor-serialization-kotlinx-json:3.0.0")
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.7.1")
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-core:1.8.1")
}

kotlin {
    jvmToolchain(17)
}

java {
    withSourcesJar()
    withJavadocJar()
}

publishing {
    publications {
        create<MavenPublication>("maven") {
            artifactId = "sf-voice-media-kotlin"
            from(components["java"])

            pom {
                name.set("sf-voice-media-kotlin")
                description.set("Kotlin SDK for the sf-voice media API")
                url.set("https://github.com/icelab-research/raidzero")
                licenses {
                    license {
                        name.set("MIT License")
                        url.set("https://opensource.org/licenses/MIT")
                    }
                }
                developers {
                    developer {
                        id.set("sf-voice")
                        name.set("sf-voice")
                    }
                }
                scm {
                    connection.set("scm:git:https://github.com/icelab-research/raidzero.git")
                    developerConnection.set("scm:git:ssh://git@github.com:icelab-research/raidzero.git")
                    url.set("https://github.com/icelab-research/raidzero")
                }
            }
        }
    }

    repositories {
        maven {
            name = "localRelease"
            url = layout.buildDirectory.dir("release-repository").get().asFile.toURI()
        }
    }
}
