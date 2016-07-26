## Gmail mail categorizer using neural word embeddings/paragraph vectors trained from Quora/Medium (crawlers are included)

### For the client:
- You need Google Go
- And the following modules (install with go get -u MODULENAME): 
  * "github.com/pkg/browser"
  * "golang.org/x/net/context"
  * "golang.org/x/net/html"
  * "golang.org/x/oauth2"
  * "golang.org/x/oauth2/google"
  * "google.golang.org/api/gmail/v1"
  * "github.com/jteeuwen/go-pkg-xmlx"
- Build with "go build"
- Run "mail-classifier.exe"
- Go to http://localhost:8080

### For the classification server:
- You need the latest JDK, Maven (https://maven.apache.org/) and IntelliJ
- Open the project with "File/New/Project from Existing Sources", select the root folder
- Run the server class, note that the training data has to be in the same folder (under the folder "trainingData") - just copy it from the client (or place both in the same folder)
- The server will now start training (takes around 10-15 minutes) and is then waiting for classification requests from the client
