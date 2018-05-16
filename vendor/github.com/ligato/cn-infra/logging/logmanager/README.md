# Log Manager

Log manager plugin allows to view and modify log levels of loggers using REST API.

**API**
- List all registered loggers:

    ```curl -X GET http://<host>:<port>/log/list```
- Set log level for a registered logger:
   ```curl -X PUT http://<host>:<port>/log/<logger-name>/<log-level>```
 
   `<log-level>` is one of `debug`,`info`,`warning`,`error`,`fatal`,`panic`
   
`<host>` and `<port>` are determined by configuration of rest.Plugin.
 