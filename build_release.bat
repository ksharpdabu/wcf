@echo off 
set  root_dir=%cd%
cd wcf 
call build.bat
cd %root_dir%
wsl rm releases -rf
wsl mkdir releases
wsl tar -czf ./releases/local-releases_`date +%%Y_%%m_%%d_%%H_%%M_%%S`.tar.gz wcf/src/wcf/cmd/local/local_*
wsl tar -czf ./releases/server-releases_`date +%%Y_%%m_%%d_%%H_%%M_%%S`.tar.gz wcf/src/wcf/cmd/server/server_*