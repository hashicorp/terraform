/*
Rabbit Hole is a Go client for the RabbitMQ HTTP API.

All HTTP API operations are accessible via `rabbithole.Client`, which
should be instantiated with `rabbithole.NewClient`.

        // URI, username, password
        rmqc, _ = NewClient("http://127.0.0.1:15672", "guest", "guest")

Getting Overview

        res, err := rmqc.Overview()

Node and Cluster Status

        var err error

        // => []NodeInfo, err
        xs, err := rmqc.ListNodes()

        node, err := rmqc.GetNode("rabbit@mercurio")
        // => NodeInfo, err

Operations on Connections

        xs, err := rmqc.ListConnections()
        // => []ConnectionInfo, err

        conn, err := rmqc.GetConnection("127.0.0.1:50545 -> 127.0.0.1:5672")
        // => ConnectionInfo, err

        // Forcefully close connection
        _, err := rmqc.CloseConnection("127.0.0.1:50545 -> 127.0.0.1:5672")
        // => *http.Response, err

Operations on Channels

        xs, err := rmqc.ListChannels()
        // => []ChannelInfo, err

        ch, err := rmqc.GetChannel("127.0.0.1:50545 -> 127.0.0.1:5672 (1)")
        // => ChannelInfo, err

Operations on Exchanges

        xs, err := rmqc.ListExchanges()
        // => []ExchangeInfo, err

        // list exchanges in a vhost
        xs, err := rmqc.ListExchangesIn("/")
        // => []ExchangeInfo, err

        // information about individual exchange
        x, err := rmqc.GetExchange("/", "amq.fanout")
        // => ExchangeInfo, err

        // declares an exchange
        resp, err := rmqc.DeclareExchange("/", "an.exchange", ExchangeSettings{Type: "fanout", Durable: false})
        // => *http.Response, err

        // deletes individual exchange
        resp, err := rmqc.DeleteExchange("/", "an.exchange")
        // => *http.Response, err

Operations on Queues

        xs, err := rmqc.ListQueues()
        // => []QueueInfo, err

        // list queues in a vhost
        xs, err := rmqc.ListQueuesIn("/")
        // => []QueueInfo, err

        // information about individual queue
        x, err := rmqc.GetQueue("/", "a.queue")
        // => QueueInfo, err

        // declares a queue
        resp, err := rmqc.DeclareQueue("/", "a.queue", QueueSettings{Durable: false})
        // => *http.Response, err

        // deletes individual queue
        resp, err := rmqc.DeleteQueue("/", "a.queue")
        // => *http.Response, err

        // purges all messages in queue
        resp, err := rmqc.PurgeQueue("/", "a.queue")
        // => *http.Response, err

Operations on Bindings

        bs, err := rmqc.ListBindings()
        // => []BindingInfo, err

        // list bindings in a vhost
        bs, err := rmqc.ListBindingsIn("/")
        // => []BindingInfo, err

        // list bindings of a queue
        bs, err := rmqc.ListQueueBindings("/", "a.queue")
        // => []BindingInfo, err

        // declare a binding
        resp, err := rmqc.DeclareBinding("/", BindingInfo{
            Source: "an.exchange",
            Destination: "a.queue",
            DestinationType: "queue",
            RoutingKey: "#",
        })
        // => *http.Response, err

        // deletes individual binding
        resp, err := rmqc.DeleteBinding("/", BindingInfo{
            Source: "an.exchange",
            Destination: "a.queue",
            DestinationType: "queue",
            RoutingKey: "#",
            PropertiesKey: "%23",
        })
        // => *http.Response, err

Operations on Vhosts

        xs, err := rmqc.ListVhosts()
        // => []VhostInfo, err

        // information about individual vhost
        x, err := rmqc.GetVhost("/")
        // => VhostInfo, err

        // creates or updates individual vhost
        resp, err := rmqc.PutVhost("/", VhostSettings{Tracing: false})
        // => *http.Response, err

        // deletes individual vhost
        resp, err := rmqc.DeleteVhost("/")
        // => *http.Response, err

Managing Users

        xs, err := rmqc.ListUsers()
        // => []UserInfo, err

        // information about individual user
        x, err := rmqc.GetUser("my.user")
        // => UserInfo, err

        // creates or updates individual user
        resp, err := rmqc.PutUser("my.user", UserSettings{Password: "s3krE7", Tags: "management policymaker"})
        // => *http.Response, err

        // deletes individual user
        resp, err := rmqc.DeleteUser("my.user")
        // => *http.Response, err

Managing Permissions

        xs, err := rmqc.ListPermissions()
        // => []PermissionInfo, err

        // permissions of individual user
        x, err := rmqc.ListPermissionsOf("my.user")
        // => []PermissionInfo, err

        // permissions of individual user in vhost
        x, err := rmqc.GetPermissionsIn("/", "my.user")
        // => PermissionInfo, err

        // updates permissions of user in vhost
        resp, err := rmqc.UpdatePermissionsIn("/", "my.user", Permissions{Configure: ".*", Write: ".*", Read: ".*"})
        // => *http.Response, err

        // revokes permissions in vhost
        resp, err := rmqc.ClearPermissionsIn("/", "my.user")
        // => *http.Response, err
*/
package rabbithole
