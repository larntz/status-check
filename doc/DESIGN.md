# controller

## mvp

1. Controller saves/reads checks via csv file
1. Controller saves/reads results via csv file 

## post-mvp

1. Controller is backed by database
1. RestAPI/Grpc interface for front end interaction e.g., CRUD'ing checks.

# worker

## mvp

1. Store all data in memory. 
1. Check controller for assigned checks every <interval>. Controller sends full list of assigned checks.
1. Checks performed as assigned, and results are sent back to controller for processing. 

## post-mvp

1. Store data in db or filesystem or filesystem based db like sqlite.
1. Check controller for updates. Devise a way to first check if assignments have changed (timestamp of last update?), and download if there are changes. Send only diff of assignments. 
    - Possibly keep a grpc connection open for comms. Can request a download of all checks on startup/when needed.
1. Distribute regional checks across workers. 
1. Checks results are sent to a controller for processing. Possibly a distributed database that is processed by controller. 
