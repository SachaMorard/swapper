# How to deploy with Google Cloud Storage 


First, create a swapper.yml configuration file locally to describe what your cluster will do
```yaml
version: '1'

master:
  driver: gcp
  project-id: my-gcp-project
  
services:
  my-app:
    ports:
      - 80:80
    containers:
      - image: nginx
        tag: 1.16.0
```

### Setting up Google Cloud Storage Authentication

_(if you want to use an existing service account skip this part)_

To run `swapper deploy`, you must first set up authentication by creating a service account and setting an environment variable. Complete the following steps to set up authentication. For more information, see the [GCP authentication documentation](https://cloud.google.com/docs/authentication/production).

>1. Step 1: In the GCP Console, go to the [Create service account key page](https://console.cloud.google.com/iam-admin/serviceaccounts/create). Then in the Service account name field, enter a name. Add a description if necessary. Then click on *CREATE* button
>2. Step 2: From the Role list, select Storage > Storage Admin. Then click on *CONTINUE* button
>3. Step 3: Click on *+ CREATE KEY* button and choose JSON format. A JSON file that contains your key downloads to your computer.

#### Swapper conf

There is multiple ways to provide authentication credentials to swapper:

- Swapper will use the current gcloud account of your machine 
- Set the environment variable `GOOGLE_APPLICATION_CREDENTIALS` with the file path of the JSON file that contains your service account key.

### Deploy swapper.yml to your bucket

After writing the swapper.yml file locally (with the google cloud storage infos inside), deploy it to your bucket simply with this command: 
```bash
swapper deploy -f swapper.yml
```

Then, you'll just have to start nodes (on any server) with specific master address:
```bash
swapper node start --join gs://swapper-master-{project-id}/myapp.yml
```

Be careful, servers that run nodes have to have proper authentication credentials to the related Bucket
