# RAW Bullet Thinking 

1. Writing strongly typed string or enums is good but when it comes to domain specific things, it needs to be flexible. 

2. Components must be as generic and as configurable as possible. 

3. All Magic numbers must be in domain specific package constants file. 

4. Write smart structure types when we needed more control over the variables like PlanID, TenantID, APIKey etc.

5. Use Pointer return and Point receivers whenever the struct is having mutex or map variables. 

6. Each Module will do their own Init, route registration, middleware creation, startup and shutdown. And we will have a central App struct that orchestrate the startup and shutdown. It will be composed with all the services and handlers.