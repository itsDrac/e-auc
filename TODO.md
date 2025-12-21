# Product
    - [ ] Create CRUD for products.
        - [X] Add Endpoint to create the product.
        - [ ] Add endpoint to Get(read the product)
            - [X] Add endpoint to get the images linked with a product.
        - [ ] Add endpoint to update the product.
    - [ ] Create endpoint for user to bid on product.

# Cache
    - [ ] Add Service dependency for redis.

# User
    - [ ] Add user configration such as threshold, or email notificatin toggle or 
    - [ ] Add user configration in cache.
    - [ ] On Updaing user configration the configrations should be updated in redis as well.

# Discussion.
    - Should we have cache dependency in
        [ ] Server level.
        [ ] Service level.
        [ ] Both. (Majorly read only in server level and read and wright in service level).
