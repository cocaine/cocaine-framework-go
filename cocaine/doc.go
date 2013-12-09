/*
This package helps you to write golang application, which can be launched in PaaS Cocaine.
Allows your application to communicate with another applications and services.

Typical application describes some handlers for incoming events. You have to register handler
for event in special map before starts main eventloop. Look at this example:

		func echo(request *cocaine.Request, response *cocaine.Response) {
			inc := <-request.Read()
			response.Write(inc)
			response.Close()
		}

		func main() {
			binds := map[string]cocaine.EventHandler{
				"echo":      echo,
			}
			Worker, err := cocaine.NewWorker()
			if err != nil {
				log.Fatal(err)
			}
			Worker.Loop(binds)
		}

Incoming event with named "echo" would be handled with echo func.
Request and Response are in/out streams to/from worker.
You are able to read incoming chunks of data, associated with current session using request.Read().
To send any chunk to client use response.Write().
Finish datastream by calling response.Close().
You must close response stream at any time before the end of the handler.
*/
package cocaine
