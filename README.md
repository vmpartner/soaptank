# SOAPTANK
Tool for load testing SOAP service. 

### Options
You can set number of threads per url and limit of requests total.  
For example you set 3 url and 3 threads, 9000 requests limit, so for each url will be fired 9000 requests in 3 parallel thread, per 3000 req in thread, total for all urls will be fired 18000 requests. 

Provide your request template file by option ```--req=```, set variables place by ```?```.   
Provide your variables file for filling template by option ```--var=```, separate variables by ```?```

Set limits in config file, provide by option ```--config=```

Please look example of config in case1 folder.


### Usage example
```
soaptank --config="case1\app.conf" --req="case1\req.xml" --var="case1\vars.txt"
```