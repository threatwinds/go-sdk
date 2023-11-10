pipeline {
    agent any

    tools { go '1.21' }
    
    stages {
        stage('Test') {
            steps {
                sh 'go test -v ./...'
            }
        }

        stage('Build') {
            steps {
                sh 'go build .'
            }
        }
    }
}