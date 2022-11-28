# go-info-webserver
Is a Go app for App Platform that exposes build and runtime environment variables via HTTP.

# Paths
- `/` returns the hostname, runtime, and buildtime environment variables.
- `/envs/build/<var_key>` returns the specific build value of a build key.
- `/envs/run/<var_key>` returns the specific run value of a run key.
1
2
3
4
1
2
3
4
5
6
7
8
9
10
1
