# ComProx
Compare Proxy is a small demo application showcasing a proxy api that can be used to compare current api routes with a new api's route. One of it's example usecases could be when porting an API from language A to language B. You can mark API A as the fromAPI and API B as the toAPI. By pointing your front end apps to the proxy api, all requests will simply be proxied to API A. In the background the proxy makes the same request to API B and compares the results.

## What does it not do?
- This proxy only logs the result as a string. This is not ideal for comparing results at a later stage.
- It only compares GET/OPTION calls since all other methods will likely result in write actions in 2 api's.
