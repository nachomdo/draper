FROM amazoncorretto:11
WORKDIR /tmp
COPY . .
RUN yum -y update && yum -y install maven
RUN mvn compile assembly:single
ENTRYPOINT ["/usr/bin/java","-jar", "/tmp/target/streams.examples-0.1-jar-with-dependencies.jar"]
