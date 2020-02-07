#!/usr/bin/env python
#   Copyright (c) 2019 AT&T Intellectual Property.
#   Copyright (c) 2019 Nokia.
#
#   Licensed under the Apache License, Version 2.0 (the "License");
#   you may not use this file except in compliance with the License.
#   You may obtain a copy of the License at
#
#       http://www.apache.org/licenses/LICENSE-2.0
#
#   Unless required by applicable law or agreed to in writing, software
#   distributed under the License is distributed on an "AS IS" BASIS,
#   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#   See the License for the specific language governing permissions and
#   limitations under the License.
#

#
#   This script does have the event handler in supervisor to follow the process state. 
#   Main parent process follows the events from the supervised daemon and the child process
#   provides the process status info via HTTP server interface.
#   When any of the process state change to PROCESS_STATE_FATAL, then the http server will 
#   respond with status code 500 to indicate faulty operation, normal status code is 200 (working).
#   Set the script configuration to supervisord.conf as follow:
#
#   [eventlistener:process-state]
#   command=python process-state.py
#   autorestart=true
#   startretries=3
#   events=PROCESS_STATE
#
#   Following is the example supervisor daemon status for each process.
#
#   ver:3.0 server:supervisor serial:16 pool:process-state poolserial:16 eventname:PROCESS_STATE_FATAL len:62
#   processname:sleep-exit groupname:sleep-exit from_state:BACKOFF
#
#   Process states are: PROCESS_STATE_STARTING, PROCESS_STATE_RUNNING, PROCESS_STATE_STOPPING, 
#   PROCESS_STATE_STOPPEDPROCESS_STATE_BACKOFF, PROCESS_STATE_FATAL
#

import os
import sys
import signal
import datetime
import argparse
import psutil
import threading
import BaseHTTPServer

# needed for python socket library broken pipe WA
from multiprocessing import Process, Manager, Value, Lock

global HTTP_STATUS
global PROCESS_STATE
global DEBUG

TABLE_STYLE = ('<style>'
    'div { font-family: Arial, Helvetica, sans-serif; font-size:8px;}'
    'p { font-family: Arial, Helvetica, sans-serif;}'
    'h1 { font-family: Arial, Helvetica, sans-serif; font-size:30px; font-weight: bold; color:#A63434;}'
    'table, th, td { border: 1px solid black; border-collapse: collapse; font-size:12px;}'
    'th, td { padding: 3px 10px 3px 10px; text-align: left;}'
    'th.thr, td.tdr { padding: 3px 10px 3px 10px; text-align: right;}'
    'th.thc, td.tdc { padding: 3px 10px 3px 10px; text-align: center;}'
    'table#t1 tr:nth-child(even) { background-color: #eee;}'
    'table#t1 tr:nth-child(odd) { background-color: #ADC0DC;}'
    'table#t1 th { background-color: #214D8B; color: white;}</style>')

def get_pid_info(pid):
    pdata = None
    if pid is not 0:
        try:
            process = psutil.Process(pid)
            # these are the item lists
            files = process.open_files()
            # get the open files and connections and count number of fd str
            sockets = process.connections()
            descriptors = str(files)+str(sockets)
            count = descriptors.count("fd=")
            pdata = {"pid": process.pid,
                    "status": process.status(),
                    "cpu": process.cpu_percent(interval=0.2),
                    "descriptors": count,
                    "memory": process.memory_info().rss}
        except (psutil.ZombieProcess, psutil.AccessDenied, psutil.NoSuchProcess):
            pdata = None
    return pdata

def get_process_html_info():
    global PROCESS_STATE
    
    try:
        html_data = ("<table width='800px' id='t1'>"
                    "<thead><tr><th>Process</th><th>Date and time</th><th>From state</th><th>to state</th>"
                    "<th class=thc>Pid</th><th class=thc>Fds</th><th class=thc>Mem</th><th class=thc>Cpu</th></tr></thead><tbody>")
        for proc,data in PROCESS_STATE.items():
            pid = 0
            descriptors = ""
            memory_usage = ""
            cpu_usage = ""
            if data['pid'] is not None:
                pdata = get_pid_info(data['pid'])
                if pdata is not None:
                    pid = pdata['pid']
                    descriptors = str(pdata['descriptors'])
                    memory_usage = str(pdata['memory']/1024)+" Kb"
                    cpu_usage = str(pdata['cpu'])+" %"
            html_data += ('<tr>'
                            '<td>'+str(proc)+'</td>'
                            '<td>'+str(data['time'])+'</td>'
                            '<td>'+str(data['from_state'])+'</td>'
                            '<td>'+str(data['state'])+'</td>'
                            '<td class=tdr>'+str(pid)+'</td>'
                            '<td class=tdr>'+descriptors+'</td>'
                            '<td class=tdr>'+memory_usage+'</td>'
                            '<td class=tdr>'+cpu_usage+'</td>'
                          '</tr>')
    finally:
        html_data += ("</tbody></table>")

    return html_data

# responds to http request according to the process status
class myHTTPHandler(BaseHTTPServer.BaseHTTPRequestHandler):
    global HTTP_STATUS
    global REFRESH_TIME    
    global PROCESS_STATE

    # write HEAD and GET to client and then close
    def do_HEAD(s):
        s.send_response(HTTP_STATUS['code'])
        s.send_header("Server-name", "supervisor-process-stalker 1.0")
        s.send_header("Content-type", "text/html")
        s.end_headers()
        s.wfile.close()
    def do_GET(s):
        try:
            """Respond to a GET request."""
            s.send_response(HTTP_STATUS['code'])
            s.send_header("Server-name", "supervisor-process-stalker 1.0")
            s.send_header("Content-type", "text/html")
            s.end_headers()
            html_data = ("<html><head><title>supervisor process event handler</title>"+TABLE_STYLE+
                        "<meta http-equiv='refresh' content='"+str(REFRESH_TIME)+"'/></head>"
                        "<body><h1>Supervisor Process Event Handler</h1>"
                        "<div><table width='800px' id='t1'><tr><td>Status code: "+str(HTTP_STATUS['code'])+"</td></tr></table></div>"
                        "<p> </p>")
            s.wfile.write(html_data)
            html = get_process_html_info()
            s.wfile.write(html)
            s.wfile.write("</body></html>")
            s.wfile.close()
        except (IOError):
            pass
            
    # make processing silent - otherwise will mess up the event handler
    def log_message(self, format, *args):
        return        
        
def HTTPServerProcess(address, port, http_status, process_state):
    global HTTP_STATUS
    global PROCESS_STATE
    
    # copy the process status global variable
    PROCESS_STATE = process_state
    HTTP_STATUS = http_status
    
    server = BaseHTTPServer.HTTPServer
    try:
        # redirect stdout to stderr so that the HTTP server won't kill the supervised STDIN/STDOUT interface
        sys.stdout = sys.stderr
        # Create a web server and define the handler to manage the
        # incoming request
        server = server((address, port), myHTTPHandler)
        # Wait forever for incoming http requests
        server.serve_forever()

    except KeyboardInterrupt:
        write_stderr('^C received, shutting down the web server')
        server.socket.close()

def dict_print(d):
    for proc,data in d.items():
        write_stderr(str(proc))
        for key,val in data.items():
            write_stderr(str(key)+' is '+str(val))

# this is the default logging, only for supervised communication
def write_stdout(s):
    # only eventlistener protocol messages may be sent to stdout
    sys.stdout.write(s)
    sys.stdout.flush()

def write_stderr(s):
    global DEBUG
    # this can be used for debug logging - stdout not allowed
    sys.stderr.write(s)
    sys.stderr.flush()

def main():
    global REFRESH_TIME
    global DEBUG

    manager = Manager()
    # stores the process status info
    PROCESS_STATE = manager.dict()
    #HTTP_STATUS_CODE = Value('d', True)
    HTTP_STATUS = manager.dict()
    HTTP_STATUS['code'] = 200
    
    write_stderr("HTTP STATUS SET TO "+str(HTTP_STATUS['code']))
   
    # default http meta key refresh time in seconds
    REFRESH_TIME = 3

    # init the default values
    ADDRESS = "0.0.0.0"     # bind to all interfaces
    PORT = 3000             # web server listen port
    DEBUG = False           # no logging
    
    parser = argparse.ArgumentParser()
    parser.add_argument('--port', dest='port', help='HTTP server listen port, default 3000', required=False, type=int)
    parser.add_argument('--debug', dest='debug', help='sets the debug mode for logging', required=False, action='store_true')
    parser.add_argument('--address', dest='address', help='IP listen address (e.g. 172.16.0.3), default all interfaces', required=False, type=str)
    parser.add_argument('--refresh', dest='refresh', help='HTTP auto refresh time in second default is 3 seconds', required=False, type=int)
    args = parser.parse_args()

    if args.port is not None:
        PORT = args.port
    if args.address is not None:
        ADDRESS = args.address
    if args.debug is not False:
        DEBUG = True;

    # Start the http server, bind to address
    httpServer = Process(target=HTTPServerProcess, args=(ADDRESS, PORT, HTTP_STATUS, PROCESS_STATE))
    httpServer.start()

    # set the signal handler this phase
    signal.signal(signal.SIGQUIT, doExit)
    signal.signal(signal.SIGTERM, doExit)
    signal.signal(signal.SIGINT, doExit)
    signal.signal(signal.SIGCLD, doExit)
    
    while httpServer.is_alive():
        # transition from ACKNOWLEDGED to READY

        write_stdout('READY\n')
        # read header line and print it to stderr
        line = sys.stdin.readline()
        write_stderr(line)

        # read event payload and print it to stderr
        headers = dict([ x.split(':') for x in line.split() ])
        process_state = headers['eventname']
        
        if process_state == 'PROCESS_STATE_FATAL':
            write_stderr('Status changed to FATAL')
            HTTP_STATUS['code'] = 500
        
        short_state = process_state.replace('PROCESS_STATE_', '')
        length = int(headers['len'])
        data = sys.stdin.read(length)
        write_stderr(data)
        
        process = dict([ x.split(':') for x in data.split() ])
        pid = 0
        if 'pid' in process:
            pid = int(process['pid'])
        now = datetime.datetime.now()
        timestamp=str(now.strftime("%Y/%m/%d %H:%M:%S"))
        PROCESS_STATE[process['processname']] = {'time': timestamp, 'state': short_state, 'from_state': process['from_state'],
                                                 'pid':pid}
        # transition from READY to ACKNOWLEDGED
        write_stdout('RESULT 2\nOK')
    httpServer.join()

def kill_child_processes():
    procs = psutil.Process().children()
    # send SIGTERM
    for p in procs:
        try:
            p.terminate()
        except psutil.NoSuchProcess:
            pass

def doExit(signalNumber, frame):
    write_stderr("Got signal: "+str(signalNumber)+" need to exit ...")
    kill_child_processes()
    exit(0)

if __name__ == '__main__':
    main()
    
