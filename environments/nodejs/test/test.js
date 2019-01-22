module.exports.hello = async function (context) {
    console.log("headers=", JSON.stringify(context.request.headers));
    console.log("body=", JSON.stringify(context.request.body));

    return {
        status: 200,
        body: "Hello, world !\n"
    };
}

module.exports.bye = async function (context) {
    console.log("headers=", JSON.stringify(context.request.headers));
    console.log("body=", JSON.stringify(context.request.body));

    return {
        status: 200,
        body: "bye, world !\n"
    };
}
