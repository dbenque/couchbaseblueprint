# Couchbase blueprint reader prototype

This prototype has been build to show what could be the proposal described on that page:

https://rndwww.nce.amadeus.net/confluence/display/PUHA/Couchbase+cluster+blueprints+as+rule-based


## Model

The datamodel is fully described in the file *api.go*

## Blueprint from the code

Initially it was meant to generate some dot chart on top of code examples:
* sampleHOS.go
* sampleRBOX.go

If you launch the program without any parameters like this:
```
go run *.go
```

It will generate 2 folders containing what blueprint could be:

- hos1
- RBox1

If you want to add another sample, create a new file mySample.go and modify the main.go to call your function in the section:

```
if len(os.Args) == 1 {
  gen_hos1()
  gen_RBox1()

  gen_mySample()   // <- call your function here

  return
}
```

## Dot Graph generation from blueprints

To place user in a Dev-Ops situation, you will have to write blueprints and inject them on Datacenter(s). You can use both yaml or json format, but yaml is more convinient for human edition.

Create a folder to host your blueprints. In that folder place all your blueprints: topology, xdcr and environment. Here is the RBox example:

```
// Topology
RBox/Erding.yaml    <-- Topology for Erding Datacenter
RBox/Proxy.yaml     <-- Topology for a Proxy cluster to be hosted in any remote datacenter
RBox/RBox.yaml      <-- Topology for a RBox tree to be hosted in any remote datacenter

// Topology environment:
/// Proxy box
RBox/ProxyEnv_GoogleEast.yaml   <-- Env file for Proxy in GoogleEast datacenter
RBox/ProxyEnv_GoogleWest.yaml   <-- Env file for Proxy in GoogleWest datacenter
/// RBoxes
RBox/RBoxEnv_AF.yaml            <-- Env file for an AF Rbox
RBox/RBoxEnv_LH.yaml            <-- Env file for an LH Rbox

// XDCR
RBox/XDCR.yaml            <-- Links to bring all 'stat' up to ADP + Links to push RBox data to all proxy. No env required
RBox/XDCR_airline.yaml    <-- Links to push RBox data to an airline RBox. Env required to apply filter

// XDCR environment
RBox/XDCREnv_AF.yaml      <-- Environment for AF filter customization
RBox/XDCREnv_LH.yaml      <-- Environment for LH filter customization
```
At the root of the project create a yaml file that will be used for injection of blueprints in the Datacenters. Note that the Datacenters are directly defined inside that file, by their name. For the RBox example the file named 'RBox.yaml'. Note the syntax with the '+' to apply the environment file:

```
topos:
  RBox/Erding.yaml: ["Erding"]
  RBox/Proxy.yaml+RBox/ProxyEnv_GoogleEast.yaml: ["GoogleEast"]
  RBox/Proxy.yaml+RBox/ProxyEnv_GoogleWest.yaml: ["GoogleWest"]
  RBox/RBox.yaml+RBox/RBoxEnv_AF.yaml: ["GoogleEast","GoogleWest"]
  RBox/RBox.yaml+RBox/RBoxEnv_LH.yaml: ["GoogleEast"]
xdcrs:
  RBox/XDCR.yaml: ["GoogleEast","GoogleWest","Erding"]
  RBox/XDCR_airline.yaml+RBox/XDCREnv_AF.yaml: ["GoogleEast","GoogleWest"]
  RBox/XDCR_airline.yaml+RBox/XDCREnv_LH.yaml: ["GoogleEast","GoogleWest"]
```

Now that our files are ready, let's run a simulation:
```
go run *.go RBox.yaml
```
The output is a graph with the full topology and XDCR links in a dot format. You need to post-process it to build an image:
```
go run *.go RBox.yaml | dot -Tpng > img.png
```

If you plan to repeat this several times, maybe you should directly invoke the image viewer. In my case I use 'feh':
```
rm img.png; go run *.go RBox.yaml | dot -Tpng > img.png; feh -F -Z img.png
```

Of course these last lines will hide you any error during the process. Thus you may a hve to run the basics commands to spot any error.

## Classic error in blueprints

We are using Dot to process the graph. Dot comes with some restriction on object name. You should not use whitespaces ' ' or tabs and dash '-'. Prefer using underscore "_" if you really need a separator.

## Labels cascading

Bucket are automatically decorated with Labels that represent their location in the topology:
```
Datacenter: nameOfDatacenter
ClusterGroup: nameOfClusterGroup_peakToken
Cluster:  nameOfCluster_nameOfInstance
```
