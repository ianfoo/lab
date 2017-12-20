package cmd

import (
	"fmt"
	"log"

	termbox "github.com/nsf/termbox-go"
	"github.com/spf13/cobra"
	"github.com/zaquestion/gocui"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

// ciCmd represents the ci command
var ciCmd = &cobra.Command{
	Use:   "ci",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		remote, _, err := parseArgsRemote(args)
		if err != nil {
			log.Fatal(err)
		}
		if remote == "" {
			remote = forkedFromRemote
		}

		// See if we're in a git repo or if global is set to determine
		// if this should be a personal snippet
		rn, err := git.PathWithNameSpace(remote)
		if err != nil {
			log.Fatal(err)
		}
		project, err := lab.FindProject(rn)
		if err != nil {
			log.Fatal(err)
		}
		sha, err := git.Sha("HEAD")
		if err != nil {
			log.Fatal(err)
		}

		g, err := gocui.NewGui(gocui.OutputNormal)
		if err != nil {
			log.Panicln(err)
		}
		defer g.Close()

		g.SetManagerFunc(jobsLayout(project.ID, sha))

		if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
			log.Panicln(err)
		}

		if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
			log.Panicln(err)
		}
	},
}

func jobsLayout(pid interface{}, sha string) func(*gocui.Gui) error {
	jobs, err := lab.CIJobs(pid, sha)
	if err != nil {
		log.Fatal(err)
	}
	return func(g *gocui.Gui) error {
		maxX, maxY := g.Size()
		var (
			stages    = 0
			lastStage = ""
		)
		// get the number of stages
		for _, j := range jobs {
			if j.Stage != lastStage {
				stages++
			}
		}
		lastStage = ""
		var (
			rowIdx   = 0
			stageIdx = 0
		)
		for _, j := range jobs {
			// The scope of jobs to show, one or array of: created, pending, running,
			// failed, success, canceled, skipped; showing all jobs if none provided
			if j.Stage != lastStage {
				rowIdx = 0
				stageIdx++
				lastStage = j.Stage
				if v, err := g.SetView("stage-"+j.Stage,
					maxX*stageIdx/(stages+1)-7, maxY/2-4,
					maxX*stageIdx/(stages+1)+7, maxY/2-2); err != nil {
					if err != gocui.ErrUnknownView {
						return err
					}
					fmt.Fprintln(v, j.Stage)
				}
			} else {
				rowIdx++
			}
			if v, err := g.SetView("jobs-"+j.Name,
				maxX*stageIdx/(stages+1)-7, maxY/2+(rowIdx*6),
				maxX*stageIdx/(stages+1)+7, maxY/2+2+(rowIdx*6)); err != nil {
				if err != gocui.ErrUnknownView {
					return err
				}
				var statChar rune
				switch j.Status {
				case "success":
					v.FgColor = gocui.ColorGreen
					statChar = '✔'
				case "failed":
					v.FgColor = gocui.ColorRed
					statChar = '✘'
				case "running":
					v.FgColor = gocui.ColorBlue
					statChar = '●'
				case "pending":
					v.FgColor = gocui.ColorYellow
					statChar = '●'
				}
				retryChar := '⟳'
				_ = retryChar
				fmt.Fprintf(v, "%c %s\n", statChar, j.Name)
			}
		}
		for i, k := 0, 1; k < len(jobs); i, k = i+1, k+1 {
			v1, err := g.View("jobs-" + jobs[i].Name)
			if err != nil {
				return err
			}
			v2, err := g.View("jobs-" + jobs[k].Name)
			if err != nil {
				return err
			}
			connect(v1, v2)
		}
		return nil
	}
}

func connect(v1 *gocui.View, v2 *gocui.View) {
	x1, y1 := v1.Position()
	x2, y2 := v2.Position()

	w, h := v1.Bounding()
	dx, dy := x2-x1, y2-y1

	if dy != 0 && dx != 0 {
		hline(x1+w, y2+h/2, dx-w)
		termbox.SetCell(x1+w+3, y2+h/2, '┳', termbox.ColorDefault, termbox.ColorDefault)
		return
	}
	if dy == 0 {
		hline(x1+w, y1+h/2, dx-w)
		return
	}

	// Drawing a job in the same stage
	// left of view
	termbox.SetCell(x2-3, y1+h/2, '┳', termbox.ColorDefault, termbox.ColorDefault)

	termbox.SetCell(x2-1, y2+h/2, '━', termbox.ColorDefault, termbox.ColorDefault)
	termbox.SetCell(x2-2, y2+h/2, '━', termbox.ColorDefault, termbox.ColorDefault)
	termbox.SetCell(x2-3, y2+h/2, '┗', termbox.ColorDefault, termbox.ColorDefault)

	vline(x2-3, y1+h, dy-1)
	vline(x2+w+3, y1+h, dy-1)

	// right of view

	termbox.SetCell(x2+w+1, y2+h/2, '━', termbox.ColorDefault, termbox.ColorDefault)
	termbox.SetCell(x2+w+2, y2+h/2, '━', termbox.ColorDefault, termbox.ColorDefault)
	termbox.SetCell(x2+w+3, y2+h/2, '┛', termbox.ColorDefault, termbox.ColorDefault)
}

func hline(x, y, l int) {
	for i := 0; i < l; i++ {
		termbox.SetCell(x+i, y, '━', termbox.ColorDefault, termbox.ColorDefault)
		//termbox.SetCell(start+i, y1+h1/2, '-', termbox.ColorDefault, termbox.ColorDefault)
	}
}

func vline(x, y, l int) {
	for i := 0; i < l; i++ {
		termbox.SetCell(x, y+i, '┃', termbox.ColorDefault, termbox.ColorDefault)
		//termbox.SetCell(x1+w1/2, start+i, '|', termbox.ColorDefault, termbox.ColorDefault)
	}
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}
func init() {
	RootCmd.AddCommand(ciCmd)
}
