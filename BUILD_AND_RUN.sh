#Build the containers
docker build --tag 660046669/bluebook:latest BlueBook

docker build --tag 660046669/mta:latest MTA

docker build --tag 660046669/msa:latest MSA

#Build the network
docker network prune

docker network create --subnet 192.168.1.0/24 emailnet-660046669

#Run the Bluebook
docker container rm -f BlueBook

#Bluebook NEEDS to be assigned 192.168.1.3
docker run --name BlueBook --net emailnet-660046669 --ip 192.168.1.3 --detach \
--publish 3000:8888 660046669/bluebook:latest

#Run two MTA servers with different domains
docker container rm -f MTA-here.com 
docker container rm -f MTA-there.com 
docker container rm -f MTA-other.org

#No need to provide an IP address! This is handled by the agent
docker run --name MTA-here.com --net emailnet-660046669 --detach --publish 3001:8888 \
660046669/mta:latest here.com

docker run --name MTA-there.com --net emailnet-660046669 --detach --publish 3002:8888 \
660046669/mta:latest there.com

docker run --name MTA-other.org --net emailnet-660046669 --detach --publish 3003:8888 \
660046669/mta:latest other.org

#Run 5 MSA clients with different addresses
docker container rm -f MSA-billgates@here.com 
docker container rm -f MSA-alanturing@here.com
docker container rm -f MSA-stevejobs@there.com 
docker container rm -f MSA-adalovelace@there.com
docker container rm -f MSA-gracehopper@other.org

#No need to provide an IP address! This is handled by the agent
docker run --name MSA-billgates.at.here.com --net emailnet-660046669 --detach --publish 3004:8888 660046669/msa:latest billgates@here.com

docker run --name MSA-alanturing.at.here.com --net emailnet-660046669 --detach --publish 3005:8888 660046669/msa:latest alanturing@here.com

docker run --name MSA-stevejobs.at.there.com --net emailnet-660046669 --detach --publish 3006:8888 660046669/msa:latest stevejobs@there.com

docker run --name MSA-adalovelace.at.there.com --net emailnet-660046669 --detach --publish 3007:8888 660046669/msa:latest adalovelace@there.com

docker run --name MSA-gracehopper.at.other.org --net emailnet-660046669 --detach --publish 3008:8888 660046669/msa:latest gracehopper@other.org

#Here are some examples...

#Sending an email:
#curl -v --request POST 'http://localhost:3006/email' --data-raw '{ "sender": "stevejobs@there.com", "receiver": "billgates@here.com", "object": "Successes and failures", "message": "Hey, remember Windows Phone? Me neither" }'

#Reading the inbox:
#curl -v --request GET 'http://localhost:3004/email/inbox'

#Read one message in the inbox:
#curl -v --request GET 'http://localhost:3004/email/inbox/{uuid}
#Find the UUID by first listing all messages in the inbox, and replace it in the URL

#Delete a message in the inbox:
#curl -v --request DELETE ' http://localhost:3004/email/inbox/{uuid}
#Find the UUID by first listing all messages in the inbox, and replace it in the URL

