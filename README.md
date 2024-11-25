# Project Github/TF/CI Dashboard

This project is a simple web application that displays information about repositories, recent commits, deployments, and Terraform builds. It provides a dashboard-like interface for teams to monitor the status of their projects and infrastructure.


### Features

- Repository List: Displays all repositories with links to CI pipelines.
- Recent Deployments: Shows details of recent deployments, including environment and stack information.
- Recent Commits: Lists the latest commits with PR links, authors, and dates.
- Terraform Build Status: Fetches and displays recent Terraform builds with their statuses.


## Installation

1. Create som access tokens :

    a. Github token with read permissions
    
    b. Terraform

2. Set up your environment variables

```
cp .env.example .env
```

And fill it with the repos you want to watch and your token.
