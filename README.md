# qlfu

Automated crud restful API on top of ql database.

This allows you to have a quick , fun way to come up with good schema
to use for ql database.

# Caveats

- This is not/ was not intended to be used in production. Stay safe don't try this at work.

- Don't make concurrent requests. The `POST /schema` endpoint messes up with the dynamic dispatched endpoint. Please make one API request at a time, this is a better way to use this. PR are welcome though.

- Starting the server, adding new schema result in a new database. There os no way/intention to reuse the same database. Isn't this fun uh!
# features

- [x] json objects to database schema. Just your normal json objects are
auto magically converted to possible ql schema.
- [x] automatic generation of crud restful json api for the generated schema. 
- [x] seamless building of relationships.One to one, one to many and many to many.
- [x] comprehensive examples/documentation
- [x] fun . ql is fun, experiment with it and hopefully you will enjoy it as much as I did. 
- [x] Auto increment id. We use a nice trick to give you unique auto increment id
- [X] one to one relationship
- [ ] one to many relationship
- [ ] many to many relationship

# Installation
    go get github.com/gernest/qlfu

This is a command line application, meaning if you did setup properly your `GOPATH` and added `$GOPATH/bin` to your system `PATH` then the binary `qlfu` should be available in your shell.

# [short] Usage

```shell
NAME:
   qlfu - magic crud and restful api for experimenting with ql database

USAGE:
   qlfu [global options] command [command options] [arguments...]
   
VERSION:
   0.1.0
   
AUTHOR(S):
   Geofrey Ernest <geofreyernest@live.com> 
   
COMMANDS:
     serve    automated crud & resful api  on ql database
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h     show help
   --version, -v  print the version
```

# [long] usage

<details>
<summary>starting the server</summary>
<code>qlfu serve --dir mydb</code>
<p>Here mydb is the directory which will store ql database files, temporary files wal etc. This directory will be created if it doesn't exist yet.</p>
</details>

<details>
<summary>creating schema</summary>
<pre><code>curl -XPOST -H &quot;Content-type: application/json&quot; -d '{
    &quot;user&quot;: {
        &quot;username&quot;: &quot;gernest&quot;,
        &quot;email&quot;: &quot;gernest@example.com&quot;,
        &quot;profile&quot;: {
            &quot;country&quot;: &quot;Tanzania&quot;,
            &quot;created_at&quot;: &quot;Mon Jan 2 15:04:05 2006&quot;,
            &quot;updated_at&quot;: &quot;Mon Jan 2 15:04:05 2006&quot;
        },
        &quot;created_at&quot;: &quot;Mon Jan 2 15:04:05 2006&quot;,
        &quot;updated_at&quot;: &quot;Mon Jan 2 15:04:05 2006&quot;
    }
}' 'http://localhost:8090/schema'
</code></pre>

<p>The objects are normal json objects. The properties of the top level objects will be considered as models from which to build the schema.</p>

<p>Object properties of type <code>number</code> will be mapped to <code>fload64</code> . You can also have time fields which a string representation of time, for now ANSIC formated time strings are the only one supported, they will map to <code>time</code> ql data type.</p>

</details>

<details>
<summary>viewing the generated schema</summary>
<pre><code>curl -XGET 'http://localhost:8090/schema'</code></pre>

<p>which gives you</p>
<pre><code>begin transaction;
   create table profiles (
    id         int64,
    country    string,
    created_at time,
    updated_at time);
   create table users (
    profiles_id int64,
    id          int64,
    username    string,
    email       string,
    created_at  time,
    updated_at  time);
commit;

</code></pre>
</details>

<details>
<summary>display the generated api</summary>
<pre><code>curl -XGET 'http://localhost:8090/v1'</code></pre>

<p> giving you </p>
<pre><code>{
  &quot;version&quot;: &quot;1&quot;,
  &quot;Endpoints&quot;: [
    {
      &quot;path&quot;: &quot;/profiles&quot;,
      &quot;params&quot;: null,
      &quot;method&quot;: &quot;post&quot;,
      &quot;payload&quot;: &quot;{\&quot;country\&quot;:\&quot;country\&quot;,\&quot;created_at\&quot;:\&quot;Thu Mar  9 11:55:14 2017\&quot;,\&quot;updated_at\&quot;:\&quot;Thu Mar  9 11:55:14 2017\&quot;}&quot;
    },
    {
      &quot;path&quot;: &quot;/profiles&quot;,
      &quot;params&quot;: null,
      &quot;method&quot;: &quot;get&quot;,
      &quot;payload&quot;: &quot;&quot;
    },
    {
      &quot;path&quot;: &quot;/profiles/:id&quot;,
      &quot;params&quot;: [
        {
          &quot;name&quot;: &quot;id&quot;,
          &quot;type&quot;: &quot;int64&quot;,
          &quot;desc&quot;: &quot;the id of profiles object&quot;,
          &quot;default&quot;: 1
        }
      ],
      &quot;method&quot;: &quot;get&quot;,
      &quot;payload&quot;: &quot;&quot;
    },
    {
      &quot;path&quot;: &quot;/users&quot;,
      &quot;params&quot;: null,
      &quot;method&quot;: &quot;post&quot;,
      &quot;payload&quot;: &quot;{\&quot;created_at\&quot;:\&quot;Thu Mar  9 11:55:14 2017\&quot;,\&quot;email\&quot;:\&quot;email\&quot;,\&quot;profiles_id\&quot;:1,\&quot;updated_at\&quot;:\&quot;Thu Mar  9 11:55:14 2017\&quot;,\&quot;username\&quot;:\&quot;username\&quot;}&quot;
    },
    {
      &quot;path&quot;: &quot;/users&quot;,
      &quot;params&quot;: null,
      &quot;method&quot;: &quot;get&quot;,
      &quot;payload&quot;: &quot;&quot;
    },
    {
      &quot;path&quot;: &quot;/users/:id&quot;,
      &quot;params&quot;: [
        {
          &quot;name&quot;: &quot;id&quot;,
          &quot;type&quot;: &quot;int64&quot;,
          &quot;desc&quot;: &quot;the id of users object&quot;,
          &quot;default&quot;: 1
        }
      ],
      &quot;method&quot;: &quot;get&quot;,
      &quot;payload&quot;: &quot;&quot;
    }
  ]
}
</code></pre>
</details>

<details>
<summary>create a user with profile</summary>
<pre><code>curl -XPOST -H &quot;Content-type: application/json&quot; -d '{&quot;username&quot;: &quot;gernest&quot;,&quot;profile&quot;:{&quot;country&quot;:&quot;Tanzania&quot;}}' 'http://localhost:8090/v1/users'
</code></pre>

<p> giving you </p>
<pre><code>{&quot;id&quot;:2,&quot;profile&quot;:{&quot;country&quot;:&quot;Tanzania&quot;,&quot;id&quot;:1},&quot;profiles_id&quot;:1,&quot;username&quot;:&quot;gernest&quot;}
</code></pre>
</details>

<details>
<summary>get a list of all users</summary>
<pre><code>curl -XGET 'http://localhost:8090/v1/users'
</code></pre>
<p> which will give you </p>
<pre><code>[{&quot;created_at&quot;:null,&quot;email&quot;:null,&quot;id&quot;:2,&quot;profiles_id&quot;:1,&quot;updated_at&quot;:null,&quot;username&quot;:&quot;gernest&quot;}]
</code></pre>
</details>

<details>
<summary>get a user by id</summary>
<pre><code>curl -XGET 'http://localhost:8090/v1/users/2'
</code></pre>
<p> which will give you </p>
<pre><code>[{&quot;created_at&quot;:null,&quot;email&quot;:null,&quot;id&quot;:2,&quot;profiles_id&quot;:1,&quot;updated_at&quot;:null,&quot;username&quot;:&quot;gernest&quot;}]
</code></pre>
</details>


# TODO

- [ ] populate timestamp fields i.e `created_at` and `updated_at`
- [ ] support one to many relationship
- [ ] support many to many relationship
- [ ] improve documentation