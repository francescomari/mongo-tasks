# MongoDB Tasks

This repository shows how to model async task processing using MongoDB as a task
queue. The project is composed of:

- An [API server](cmd/api/main.go), that exposes an HTTP API to submit tasks and
  retrive the status of a task. The API server saves the task in MongoDB for the
  workers to pick them up.
- A [worker](cmd/worker/main.go), that polls MongoDB for new tasks and executes
  them. The worker sleeps for a certain amount of time and updates the status of
  the task on start and finish.
- A [client](cmd/client/main.go), that continuously submits new tasks to the API
  server and polls for the task until it is completed.

## Run

You need Docker and Docker Compose to run the project. To start a deployment
with MongoDB, the API server, and five replicas each of client and worker, run:

    docker-compose up

After Docker Compose builds the images and starts the containers, you should see
an output similar to the following:

    api_1     | 2021/04/13 13:22:37 connected to MongoDB
    worker_1  | 2021/04/13 13:22:38 connected to MongoDB
    worker_3  | 2021/04/13 13:22:37 connected to MongoDB
    worker_2  | 2021/04/13 13:22:37 connected to MongoDB
    worker_4  | 2021/04/13 13:22:38 connected to MongoDB
    worker_5  | 2021/04/13 13:22:37 connected to MongoDB
    client_5  | 2021/04/13 13:22:43 submitted task 57e1c9ed-0537-4cbf-83b6-0bae086b4c45
    client_4  | 2021/04/13 13:22:43 submitted task 57ed60a1-0dbf-4881-9f0d-bd7dd990f601
    client_1  | 2021/04/13 13:22:43 submitted task c5b68587-bb93-4c3b-a2bb-75d59e804dd6
    client_3  | 2021/04/13 13:22:43 submitted task 146e6e47-eb2c-406c-9191-bfeac7a818ba
    client_2  | 2021/04/13 13:22:43 submitted task 8ebc7db3-0fb9-456b-8f9c-eced6c93a9c5
    worker_5  | 2021/04/13 13:22:43 found task 57e1c9ed-0537-4cbf-83b6-0bae086b4c45 created at 2021-04-13 13:22:43.141 +0000 UTC with data {"random":"e5369f9c-a929-47b2-b259-a8c6923c36ae"}
    worker_2  | 2021/04/13 13:22:43 found task 57ed60a1-0dbf-4881-9f0d-bd7dd990f601 created at 2021-04-13 13:22:43.171 +0000 UTC with data {"random":"6e32bc05-99a6-45e2-aa91-30bfcec23707"}
    worker_3  | 2021/04/13 13:22:43 found task c5b68587-bb93-4c3b-a2bb-75d59e804dd6 created at 2021-04-13 13:22:43.187 +0000 UTC with data {"random":"b813eb1c-ac19-48b1-842a-7ae635d48915"}
    worker_1  | 2021/04/13 13:22:44 found task 146e6e47-eb2c-406c-9191-bfeac7a818ba created at 2021-04-13 13:22:43.197 +0000 UTC with data {"random":"e0e71297-351c-451a-bb18-68ab8a9a232b"}
    worker_4  | 2021/04/13 13:22:44 found task 8ebc7db3-0fb9-456b-8f9c-eced6c93a9c5 created at 2021-04-13 13:22:43.211 +0000 UTC with data {"random":"3e530cbc-8c8c-438d-a30d-bfb4f38c319f"}
    client_5  | 2021/04/13 13:22:58 task 57e1c9ed-0537-4cbf-83b6-0bae086b4c45 terminated on 2021-04-13 13:22:53.814 +0000 UTC
    client_4  | 2021/04/13 13:22:58 task 57ed60a1-0dbf-4881-9f0d-bd7dd990f601 terminated on 2021-04-13 13:22:53.819 +0000 UTC
    client_1  | 2021/04/13 13:22:58 task c5b68587-bb93-4c3b-a2bb-75d59e804dd6 terminated on 2021-04-13 13:22:53.956 +0000 UTC
    client_3  | 2021/04/13 13:22:58 task 146e6e47-eb2c-406c-9191-bfeac7a818ba terminated on 2021-04-13 13:22:54.044 +0000 UTC
    client_2  | 2021/04/13 13:22:58 task 8ebc7db3-0fb9-456b-8f9c-eced6c93a9c5 terminated on 2021-04-13 13:22:54.094 +0000 UTC
