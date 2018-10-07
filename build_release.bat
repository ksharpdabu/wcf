@echo off 
set  cur=%cd%
cd wcf 
build.bat
cd %cur%
wsl rm releases -rf
wsl mkdir releases
wsl tar -czf ./releases/local-releases_`date +%%Y_%%m_%%d_%%H_%%M_%%S`.tar.gz wcf/src/wcf/cmd/local/local_*
wsl tar -czf ./releases/server-releases_`date +%%Y_%%m_%%d_%%H_%%M_%%S`.tar.gz wcf/src/wcf/cmd/server/server_*