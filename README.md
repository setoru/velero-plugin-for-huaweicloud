# Velero Deployment

## Overview

This plugin is used for saving and retrieving backup data on Huawei Cloud OBS as an object storage plugin. The backup includes metadata files of Kubernetes resources and CSI objects, as well as the progress of asynchronous operations. It is also used to store result data from backups and restores, including log files, warning/error files, and more.

## Prerequisites

- CCE cluster
- Git
- Docker
- Docker-buildx

## Deployment

1. Log in to a node in the CCE cluster and connect to the cluster.

2. Install the Velero CLI tool.

   Download `velero-v1.15.0-linux-amd64.tar.gz` from the official website and upload it to the node.

   Extract Velero:

   ```sh
   tar -xvf velero-v1.15.0-linux-amd64.tar.gz
   mv velero-v1.15.0-linux-amd64/velero /usr/local/bin/velero
	```

3. Build the Velero plugin image.

    Clone the Velero plugin repository:

   ```sh
   git clone https://github.com/setoru/velero-plugin-for-huaweicloud.git
	```

   Build the image:

   ```sh
   cd velero-plugin-for-huaweicloud
   make container
   ```

4. Create a new OBS bucket. The bucket should not contain other directories.

5. Create a credentials configuration file.

   ```sh
   mkdir velero
   vi /home/velero/credentials-velero
   ```

   Add the following content:

   ```sh
   OBS_ACCESS_KEY=<Your AK>
   OBS_SECRET_KEY=<Your SK>
   ```

6. Install Velero using the CLI tool.

   ```sh
   velero install \
       --provider huawei.com/huaweicloud \
       --plugins swr.ap-southeast-1.myhuaweicloud.com/huaweiclouddeveloper/velero-plugin-for-huaweicloud:v1.0.0 \
       --bucket <Your Bucket> \
       --secret-file /home/velero/credentials-velero \
       --backup-location-config \
   endpoint=<Your Endpoint>
   ```

   Replace `<Your Bucket>` and `<Your Endpoint>` with your bucket name and endpoint.

   Enter the CCE cluster, upgrade the Velero workload, replace the image with the one uploaded to SWR, and add the environment variable `HUAWEI_CLOUD_CREDENTIALS_FILE=/credentials/cloud`.

7. Check the storage location status.

   ```sh
   velero backup-location get
   ```

   `available` indicates a successful installation.

8. Backup data.

    Backup all resources in the `default` namespace:

   ```sh
   velero backup create default-backup --include-namespaces default
   ```

   Check if the backup was successful:

   ```sh
   velero backup get
   ```

   `completed` indicates the backup was successful.

   View details and logs:

   ```sh
   velero backup describe default-backup
   velero backup logs default-backup
   ```

9. Delete resources in the `default` namespace.

10. Restore resources.

    Update the backup storage location to read-only mode:

    ```sh
    kubectl patch backupstoragelocation default \
        --namespace velero \
        --type merge \
        --patch '{"spec":{"accessMode":"ReadOnly"}}'
    ```

    Restore resources using the created backup:

    ```sh
    velero restore create --from-backup default-backup
    ```

    Check the restore status:

    ```sh
    velero restore get
    ```

    `completed` indicates a successful restore.

    Restore the backup storage location to read-write mode:

    ```sh
    kubectl patch backupstoragelocation default \
        --namespace velero \
        --type merge \
        --patch '{"spec":{"accessMode":"ReadWrite"}}'
    ```

11. Delete a backup.

    ```sh
    velero backup delete default-backup
    ```

12. Uninstall Velero resources.

    ```sh
    kubectl delete namespace/velero clusterrolebinding/velero
    kubectl delete crds -l component=velero
    ```
