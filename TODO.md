# Product
    - [ ] Create CRUD for products.
        - [X] Add Endpoint to create the product.
        - [ ] Add endpoint to Get(read the product)
            - [X] Add endpoint to get the images linked with a product.
        - [ ] Add endpoint to update the product.
    - [ ] Create endpoint for user to bid on product.

# Cache
    - [X] Add Service dependency for redis.

# User
    - [ ] Add user configration such as threshold, or email notificatin toggle or 
    - [ ] Add user configration in cache.
    - [ ] On Updaing user configration the configrations should be updated in redis as well.

# Swagger
    - [X] Add Swagger support for the project.
    - [X] Add docs for health
    - [X] Add docs for Create, Login and Profile user.
        
# Known Issue
    - [ ] When get product image is called it shares the url which contains localhost, I dont think frontend will be able to load image using localhost. therefore we'll need to change the localhost to domain thats pointing to storage service (Or some other work around.)
    - [ ] We'll need to chagen the url params to query string params. 


# Discussion.
    - Should we have cache dependency in
        [X] Server level.
        [ ] Service level.
        [ ] Both. (Majorly read only in server level and read and wright in service level).
    - [ ] How can we use cache in authentication.
