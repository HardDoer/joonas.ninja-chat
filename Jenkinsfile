pipeline {
    agent any 
    stages {
            stage('Checkout') {
                steps {
                    checkout scm
                }
            }
            stage('Build'){
                steps{
                    sh "docker build -t jnokikana/joonas.ninja:joonas.ninja-chat ."
                }
            }
            stage('Tag'){
                steps{
                    sh "docker tag jnokikana/joonas.ninja:joonas.ninja-chat jnokikana/joonas.ninja:joonas.ninja-chat"
                }
            }
            stage('Login') {
                steps {
                    script{
                        withCredentials([usernamePassword(usernameVariable: 'USERNAME', passwordVariable: 'PASSWORD', credentialsId: '06401b2c-c73a-4fb8-9a44-95cdc301d3d3')]) {
                            sh 'docker login --username $USERNAME --password $PASSWORD'
                        }
                    }
                }
            }
            stage('Push to artifactory'){
                steps{
                    sh "docker push jnokikana/joonas.ninja:joonas.ninja-chat"
                }
            }
    }
}