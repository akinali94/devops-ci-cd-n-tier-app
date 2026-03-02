pipeline {
    agent any

    environment {
        // Short commit SHA used as the Docker image tag for traceability.
        IMAGE_TAG = "${env.GIT_COMMIT[0..6]}"
        // Go is installed at /usr/local/go by ec2-setup.sh but is not in the
        // jenkins user's default PATH, so we add it explicitly here.
        PATH = "/usr/local/go/bin:${env.PATH}"
    }

    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }

        stage('Test') {
            parallel {
                stage('Test api-service') {
                    steps {
                        dir('services/api-service') {
                            sh 'go test ./... -v -count=1'
                        }
                    }
                }
                stage('Test auth-service') {
                    steps {
                        dir('services/auth-service') {
                            sh 'go test ./... -v -count=1'
                        }
                    }
                }
            }
        }

        stage('Build Images') {
            parallel {
                stage('Build api-service') {
                    steps {
                        sh 'docker build -t api-service:${IMAGE_TAG} -t api-service:latest services/api-service/'
                    }
                }
                stage('Build auth-service') {
                    steps {
                        sh 'docker build -t auth-service:${IMAGE_TAG} -t auth-service:latest services/auth-service/'
                    }
                }
            }
        }

        stage('Deploy') {
            steps {
                // Source secrets from the env file on the EC2 instance.
                // This file is created manually once during EC2 setup and is never committed to Git.
                sh '''
                    set -a
                    source /home/ec2-user/app.env
                    set +a
                    IMAGE_TAG=${IMAGE_TAG} docker compose -f docker-compose.yml up -d --remove-orphans
                    docker compose -f docker-compose.yml ps
                '''
            }
        }
    }

    post {
        success {
            echo "Deploy succeeded — image tag: ${env.IMAGE_TAG}"
        }
        failure {
            echo "Pipeline failed on commit ${env.GIT_COMMIT}. Check logs above."
        }
        always {
            // Remove images older than 24 hours to keep EC2 disk usage in check.
            sh 'docker image prune -f --filter "until=24h"'
        }
    }
}
