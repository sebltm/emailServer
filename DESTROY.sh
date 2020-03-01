docker stop MSA-billgates.at.here.com MSA-alanturing.at.here.com MSA-stevejobs.at.there.com MSA-adalovelace.at.there.com MSA-gracehopper.at.other.org

docker stop MTA-here.com MTA-there.com MSA-here.com MSA-there.com

docker stop BlueBook

docker container rm -f BlueBook MTA-here.com MTA-there.com MTA-other.org

docker container rm -f MSA-billgates.at.here.com MSA-alanturing.at.here.com MSA-stevejobs.at.there.com MSA-adalovelace.at.there.com MSA-gracehopper.at.other.org

docker image rm -f 660046669/bluebook:latest 660046669/mta:latest 660046669/msa:latest 

docker network rm emailnet-660046669
