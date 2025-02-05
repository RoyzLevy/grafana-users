###### Task information

## Tools/Technologies

1. Docker
2. Minikube
3. Helm
4. Grafana API (Admin and Org): https://grafana.com/docs/grafana/latest/developers/http_api/admin/ https://grafana.com/docs/grafana/latest/developers/http_api/org/
5. Golang

## Approaches

The first approach will be to create an init container which will run as part of the Grafana pod initialization.
This approach is good in case we won't need to re-run/retry the users creation process.
In this approach, users creation will be a blocker for Grafana start up - If the init container fails the whole pod fails.

The second approach will be creating a k8s job which will have an helm hook and will run after the Grafana pod is started, But still as part of the Grafana installation.
This approach is good in case we want to be able to retry and rerun user creation without restarting the whole Grafana pod.
Also it won't run on each pod restart so we can have our users created once on helm installation and be done.

We will be using the first approach:
I've created golang code for handling the "Para" organization and the 3 users.
For all connection purposes the code uses the admin credentials provided in grafana-admin-secret.yaml (should be provided from external secret store)
The code handles checking if the org exists and if not - creates it.
Then it continues to create the users with the information that I provide in users-list.yaml configmap.
For the last step - the modifyUserRole function takes care of assigning the users to the org that we created with the correct roles (Viewer, Editor and Admin)
I used golang and not bash script so we could have more functionalities in the future and better error handling in the users provisioning process.

The application is dockerized using the Dockerfile that I've created.
It compiles the code and uses multistaging in order to have slimmer container for our app.
It also uses strict versioning and not latest as a good practice.
This docker image is built locally and then loaded to our minikube local k8s cluster.

values.yaml is the file configuring our helm chart grafana installation and specifies:

- version of grafana that we want to use.
- persistence so we can have the data saved even if the grafana pod restarts.
- extraContainers for our sidecar provision container that uses our dockerized app.
- extraContainerVolumes for our user list mounting to the init container.

## Improvements

The code does not handle updating user roles. Right now the user provision fails if a user already exists.
The init container keeps restarting and I did not find a way to have a Job functionality in the built in parameters the the values.yaml of the grafana helm chart receives (https://github.com/grafana/helm-charts/tree/main/charts/grafana)

## Handling more users and sensitive information

Right now we're using a ConfigMap. This can easily scale to 50 users and beyond, as ConfigMaps in Kubernetes can hold large amounts of data (up to 1MB by default).
If we will anticipate scaling further, to thousands of users, we can consider storing the users in a more scalable format, such as an external database like postgres.
I put the sensitive information (admin user/password and user passwords) in the repo for simplification purposes. In any other real usecase we would handle sensitive information in an external secret registry like Hashicorp Vault.

###### Installation manual

## Prerequisites

1. Docker
2. Minikube
3. kubectl
4. Helm
5. Golang

## Installation and verification

run ./run-and-provision.sh script

wait for a few minutes to let grafana initiate and portforwarding to run
log in to localhost:3000 and provide admin user/password: admin,mypassword
in the menu panel go to Administration -> General -> Organizations
check the "Para" organization for the created users and their roles
